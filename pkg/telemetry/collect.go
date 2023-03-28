package telemetry

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	if os.Getenv(DisabledEnvVar) != "true" {
		Collector = NewDefaulCollector()
	}
}

var (
	Collector EventCollector = &DefaultCollector{
		enabled: false,
	}
)

const (
	eventsCountThreshold = 10          //dev //TODO: replace with proper value - 1000 ?
	minUploadInterval    = time.Minute //dev //TODO: replace with proper value - 60 * time.Minute
)

type EventCollector interface {
	IsEnabled() bool
	// RecordEvent adds the produced event to a buffer to eventually be sent to the telemetry backend
	RecordEvent(e *Event)
	// NewEvent allocates a new Event struct to be populated by the caller.
	NewEvent(t EventType) *Event
}

func NewDefaulCollector() *DefaultCollector {
	hostClient, err := client.New(ctrl.GetConfigOrDie(), client.Options{})
	if err != nil {
		panic(err)
	}

	vclusterNamespace, err := clienthelper.CurrentNamespace()
	if err != nil {
		panic(err)
	}

	c := &DefaultCollector{
		enabled:           true,
		hostClient:        hostClient,
		vclusterNamespace: vclusterNamespace,
		eventsBuffer:      make([]*Event, 0, eventsCountThreshold),
		bufferMutex:       sync.Mutex{},
		eventsToUpload:    make([]*Event, 0, eventsCountThreshold), // preallocating this slice as well because it will be swapped with data slice
		uploadMutex:       sync.Mutex{},
		nextUploadTime:    time.Now().Add(minUploadInterval),
	}

	go c.timedUploadRoutine()

	return c
}

type DefaultCollector struct {
	enabled           bool
	hostClient        client.Client
	vclusterNamespace string
	// eventsBuffer contains events that are going to be uploaded later
	eventsBuffer []*Event
	// bufferMutex controlls concurrent access to the data field
	bufferMutex sync.Mutex
	// eventsToUpload contains a slice of the Events that should be uploaded immediately.
	// After the upload the slice length is set to 0, but it should not be released
	// from the memory in order to avoid unnecessary (de)allocation of the underlying array.
	eventsToUpload []*Event
	// lastUploadTime contains the Time of the previous upload
	lastUploadTime time.Time
	// uploadMutex controlls access to the eventsToUpload slice and lastUploadTime
	uploadMutex sync.Mutex
	// nextUploadTime contains the Time of the next planned upload that should happen
	// if the events count has not reached the eventsCountThreshold.
	nextUploadTime time.Time
	// uploadMutex controlls access to the nextUploadTime
	nextUploadMutex sync.Mutex
}

func (d *DefaultCollector) IsEnabled() bool {
	return d.enabled
}

func (d *DefaultCollector) NewEvent(t EventType) *Event {
	return &Event{Type: t}
}

func (d *DefaultCollector) RecordEvent(e *Event) {
	//TODO: decide if we want to keep appending event if the threshold is already reached
	// if yes, we should also decide if we are going to deallocate the buffer if capacity > threshold
	// if not, it means we are going to drop events if the threshold was reached before we sent previous batch

	//TODO: ignore initial reconciling

	if len(d.eventsBuffer) < eventsCountThreshold {
		d.bufferMutex.Lock()
		d.eventsBuffer = append(d.eventsBuffer, e)
		// defer the Unlock in case the buffer is full - tryUpload requires Lock to be held by the caller
		defer d.bufferMutex.Unlock()
	}

	if len(d.eventsBuffer) == eventsCountThreshold {
		d.tryUpload()
	}
}

// StartUpload assumes that the caller holds the Lock for the bufferMutex
func (d *DefaultCollector) tryUpload() {
	// TryLock is used here to avoid blocking the callers in case another upload routine is
	// currently in progress, in such case the
	locked := d.uploadMutex.TryLock()
	if locked {
		// swap the eventsBuffer and eventsToUpload immediately
		d.swapBuffer()

		// execute upload without blocking
		go func() {
			d.executeUpload()
			d.uploadMutex.Unlock()
		}()
	}
}

func (d *DefaultCollector) swapBuffer() {
	tmp := d.eventsToUpload[:0] // keep the allocated memory, but set len=0
	d.eventsToUpload = d.eventsBuffer
	d.eventsBuffer = tmp
}

// timedUploadRoutine will run infinitely to ensure that the events are uploaded at least every minUploadInterval.
// timedUploadRoutine should be executed in a dedicated goroutine after creating the collector.
func (d *DefaultCollector) timedUploadRoutine() {
	for {
		fmt.Printf("timedUploadRoutine new loop iteration started\n") //dev
		d.nextUploadMutex.Lock()
		nt := d.nextUploadTime
		d.nextUploadMutex.Unlock() // Unlock immediately so the executeUpload can Lock it

		if time.Now().After(nt) {
			// Try to acquire uploadMuttex first to avoid blocking buffer unnecessarily.
			// TryLock is used to avoid executing timed upload if another upload is already in progress.
			locked := d.uploadMutex.TryLock()
			if locked {
				d.bufferMutex.Lock()
				d.swapBuffer()
				d.bufferMutex.Unlock() // intentionally not defered because to unlock ASAP
				d.executeUpload()
				d.uploadMutex.Unlock()
			}
		}
		d.nextUploadMutex.Lock()
		// Update the time of the next regular upload regardless of the update in executeUpload,
		// in case executeUpload is running in parallel and has not updated the nextUploadTime yet.
		nt = time.Now().Add(minUploadInterval)
		d.nextUploadTime = nt
		d.nextUploadMutex.Unlock() // intentionally not defered because this function never returns
		time.Sleep(time.Until(nt))
	}
}

// executeUpload assumes that the caller holds the Lock for the uploadMutex
func (d *DefaultCollector) executeUpload() {
	// update the time of the next regular upload
	d.nextUploadMutex.Lock()
	d.nextUploadTime = time.Now().Add(minUploadInterval)
	d.nextUploadMutex.Unlock()

	r := SyncerTelemetryRequest{
		Events: d.eventsToUpload,
	}
	// set TimeSinceLastUpload if this is not the first upload
	if !d.lastUploadTime.IsZero() {
		t := int(time.Since(d.lastUploadTime).Milliseconds())
		r.TimeSinceLastUpload = &t
	}
	d.lastUploadTime = time.Now()

	// call the function that will return all instance properties
	r.InstanceProperties = d.getSyncerInstanceProperties()

	marshaled, err := json.Marshal(r)
	// handle potential Marshal errors
	if err != nil {
		l := loghelper.New("telemetry")
		l.Debugf("failed to json.Marshal telemetry requests: %v", err)
		return
	}
	fmt.Printf("\n\n%s\n\n", marshaled) //dev //TODO: remove this
	//TODO: upload the data
}

func (d *DefaultCollector) getSyncerInstanceProperties() SyncerInstanceProperties {
	p := SyncerInstanceProperties{
		UID:           getSyncerUID(d.hostClient, d.vclusterNamespace)(),
		CreationType:  os.Getenv(InstanaceCreatorEnvVar),
		Arch:          runtime.GOARCH,
		OS:            runtime.GOOS,
		SyncerVersion: SyncerVersion,
	}
	// UID                      string
	// CreationType             string
	// Arch                     string
	// OS                       string
	// SyncerVersion            string
	// VirtualKubernetesVersion string // TODO: helper function to get virtualKubernetesVersion - not cached
	// HostKubernetesVersion    string // TODO: helper function to get hostKubernetesVersion - cached
	// SyncerPodsReady          int    // TODO: helper function to get syncerPodsReady- not cached
	// SyncerPodsFailing        int    // TODO: helper function to get syncerPodsFailing- not cached
	// SyncerPodCreated         int    // TODO: helper function to get syncerPodCreated- not cached
	// SyncerPodRestarts        int    // TODO: helper function to get syncerPodRestarts- not cached
	// SyncerFlags              string // TODO: function to get syncerFlags - json formatted? - cached
	// SyncerMemoryRequests     int    // TODO: use (q *Quantity) AsInt64() ? - cached
	// SyncerMemoryLimits       int    // TODO: use (q *Quantity) AsInt64() ? - cached
	// SyncerCpuRequests        int    // TODO: use (q *Quantity) AsInt64() ? - cached
	// SyncerCpuLimits          int    // TODO: use (q *Quantity) AsInt64() ? - cached
	// CachedObjectsCount       string // TODO: function to getCachedObjects - leader only, json formatted - not cached
	// VclusterServiceType      string // TODO: function to getVclusterServiceType (LoadBalancer, NodePort, etc.)  - cached

	return p
}
