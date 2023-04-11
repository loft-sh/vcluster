package telemetry

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/loft-sh/vcluster/pkg/serviceaccount"
	"gopkg.in/square/go-jose.v2/jwt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	if os.Getenv(DisabledEnvVar) != "true" {
		Collector = NewDefaultCollector()
	}

	// temporary test code for testing release workflow changes
	//TODO: remove this code
	endpointOverride := os.Getenv("SYNCER_TELEMETRY_ENDPOINT")
	if endpointOverride != "" {
		syncerTelemetryEndpoint = endpointOverride
	}
}

var (
	Collector EventCollector = &DefaultCollector{
		enabled: false,
	}

	syncerTelemetryEndpoint = "https://admin.loft.sh/analytics/v1/vcluster/v1/syncer"

	// a dummy key so this doesnt fail for dev/testing, this is set by build flag in release action
	telemetryPrivateKey string = `-----BEGIN RSA PRIVATE KEY-----
 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
 -----END RSA PRIVATE KEY-----`
)

const (
	eventsCountThreshold = 500
	maxUploadInterval    = 5 * time.Minute
	// minimum time between uploading events
	minUploadInterval = time.Minute
)

type EventCollector interface {
	IsEnabled() bool
	// RecordEvent adds the produced event to a buffer to eventually be sent to the telemetry backend
	RecordEvent(e *Event)
	// NewEvent allocates a new Event struct to be populated by the caller.
	NewEvent(t EventType) *Event
}

func NewDefaultCollector() *DefaultCollector {
	hostClient, err := client.New(ctrl.GetConfigOrDie(), client.Options{})
	if err != nil {
		panic(err)
	}

	vclusterNamespace, err := clienthelper.CurrentNamespace()
	if err != nil {
		panic(err)
	}

	decodedCertificate, err := base64.RawStdEncoding.DecodeString(telemetryPrivateKey)
	if err != nil {
		panic(err)
	}

	privateKey, err := parsePrivateKey(decodedCertificate)
	if err != nil {
		panic(err)
	}

	tokenGenerator, err := serviceaccount.JWTTokenGenerator("vcluster-telemetry", privateKey)
	if err != nil {
		panic(err)
	}

	c := &DefaultCollector{
		log:               loghelper.New("telemetry"),
		enabled:           true,
		hostClient:        hostClient,
		vclusterNamespace: vclusterNamespace,

		// events doesn't need to match eventsCountThreshold, we just
		// need to make sure its fast enough emptied.
		events: make(chan *Event, 100),
		buffer: newEventBuffer(eventsCountThreshold),

		tokenGenerator: tokenGenerator,
	}

	go c.start()

	return c
}

type DefaultCollector struct {
	log     loghelper.Logger
	enabled bool

	events      chan *Event
	buffer      *eventBuffer
	bufferMutex sync.Mutex

	hostClient        client.Client
	vclusterNamespace string

	// lastUploadTime contains the Time of the previous upload
	lastUploadTime time.Time

	tokenGenerator       serviceaccount.TokenGenerator
	token                string
	tokenLastGeneratedAt time.Time
}

func (d *DefaultCollector) IsEnabled() bool {
	return d.enabled
}

func (d *DefaultCollector) NewEvent(t EventType) *Event {
	return &Event{Type: t}
}

func (d *DefaultCollector) RecordEvent(e *Event) {
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

	// catch termination signal in order to force metrics upload
	terminate := false
	terminationChannel := make(chan os.Signal, 2)
	signal.Notify(terminationChannel, os.Interrupt, syscall.SIGTERM)

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
			if timeSinceStart < minUploadInterval {
				select {
				// wait the rest of the time here before proceeding
				case <-time.After(minUploadInterval - timeSinceStart):
				case <-terminationChannel:
					terminate = true
					fmt.Println("Termination signal") //dev
				}
			}
		case <-time.After(maxUploadInterval):
		case <-terminationChannel:
			terminate = true
			fmt.Println("Termination signal (2)") //dev
		}

		// get the currently stored events
		events := d.exchangeBuffer()
		d.executeUpload(events)

		// Exit if the upload was caused by the SIGTERM
		if terminate {
			os.Exit(1)
		}
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
	if d.token == "" || d.tokenLastGeneratedAt.Before(time.Now().Add(-time.Hour)) {
		token, err := d.tokenGenerator.GenerateToken(&jwt.Claims{}, &jwt.Claims{})
		if err != nil {
			d.log.Debugf("failed to generate telemetry request signed token: %v", err)

			return
		}

		d.token = token
	}

	r := SyncerTelemetryRequest{
		Events: buffer,
		Token:  d.token,
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
		d.log.Debugf("failed to json.Marshal telemetry request: %v", err)
		return
	}

	// send the telemetry data and ignore the response
	resp, err := http.Post(
		syncerTelemetryEndpoint,
		"multipart/form-data",
		bytes.NewReader(marshaled),
	)
	if err != nil {
		d.log.Debugf("error sending telemetry request: %v", err)
	} else if resp.StatusCode != 200 {
		d.log.Debugf("telemetry request returned non 200 status code: %v", err)
	}
}

func (d *DefaultCollector) getSyncerInstanceProperties() SyncerInstanceProperties {
	p := SyncerInstanceProperties{
		UID:                 getSyncerUID(d.hostClient, d.vclusterNamespace)(),
		InstanceCreatorType: os.Getenv(InstanaceCreatorTypeEnvVar),
		InstanceCreatorUID:  os.Getenv(InstanaceCreatorUIDEnvVar),
		Arch:                runtime.GOARCH,
		OS:                  runtime.GOOS,
		SyncerVersion:       SyncerVersion,
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
