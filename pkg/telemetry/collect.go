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

	// time to wait at least before sending an event
	minWait = time.Second * 30
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

		// events doesn't need to match eventsCountThreshold, we just
		// need to make sure its fast enough emptied.
		events: make(chan *Event, 100),
		buffer: newEventBuffer(eventsCountThreshold),
	}

	go c.start()

	return c
}

type DefaultCollector struct {
	enabled bool

	events      chan *Event
	buffer      *eventBuffer
	bufferMutex sync.Mutex

	hostClient        client.Client
	vclusterNamespace string

	// lastUploadTime contains the Time of the previous upload
	lastUploadTime time.Time
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
	d.events <- e
}

func (d *DefaultCollector) start() {
	// constantly pull events into this buffer
	go func() {
		for event := range d.events {
			d.bufferMutex.Lock()
			d.buffer.Append(event)
			d.bufferMutex.Unlock()
		}
	}()

	// constantly loop
	for {
		// either wait until buffer is full or up to 5 minutes
		startWait := time.Now()
		select {
		// we don't need to lock here for the buffer, because its only
		// exchanged below and this method can only run once at the same time
		// so this is safe.
		case <-d.buffer.Full():
			timeSinceStart := time.Since(startWait)
			if timeSinceStart < minWait {
				// wait the rest of the time here before proceeding
				time.Sleep(minWait - timeSinceStart)
			}
		case <-time.After(time.Minute * 5):
		}

		// get the currently stored events
		events := d.exchangeBuffer()
		d.executeUpload(events)
	}
}

func (d *DefaultCollector) exchangeBuffer() []*Event {
	d.bufferMutex.Lock()
	defer d.bufferMutex.Unlock()

	events := d.buffer.Get()
	d.buffer = newEventBuffer(eventsCountThreshold)
	return events
}

// executeUpload assumes that the caller holds the Lock for the uploadMutex
func (d *DefaultCollector) executeUpload(buffer []*Event) {
	r := SyncerTelemetryRequest{
		Events: buffer,
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
