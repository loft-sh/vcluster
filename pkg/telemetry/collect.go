package telemetry

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/loft-sh/vcluster/pkg/serviceaccount"
	"github.com/spf13/cobra"
	"gopkg.in/square/go-jose.v2/jwt"
	"k8s.io/client-go/kubernetes"

	vcontext "github.com/loft-sh/vcluster/cmd/vcluster/context"
	"github.com/loft-sh/vcluster/pkg/telemetry/types"
	"github.com/loft-sh/vcluster/pkg/util/clienthelper"
	"github.com/loft-sh/vcluster/pkg/util/loghelper"
	ctrl "sigs.k8s.io/controller-runtime"
)

func init() {
	// temporary test code for testing release workflow changes
	endpointOverride := os.Getenv("SYNCER_TELEMETRY_ENDPOINT")
	if endpointOverride != "" {
		syncerTelemetryEndpoint = endpointOverride
	}
	//TODO: remove the code above

	c := types.SyncerTelemetryConfig{}
	if os.Getenv(ConfigEnvVar) != "" {
		err := json.Unmarshal([]byte(os.Getenv(ConfigEnvVar)), &c)
		if err != nil {
			loghelper.New("telemetry").Infof("failed to parse telemetry config from the %s environment variable: %v", ConfigEnvVar, err)
		}
	}
	if c.Disabled == "true" {
		Collector = &DefaultCollector{
			enabled: false,
		}
		return
	}
	var err error
	Collector, err = NewDefaultCollector(context.Background(), c)
	if err != nil {
		// Log the problem but don't fail - use disabled Collector instead
		loghelper.New("telemetry").Infof("%s", err.Error())
		Collector = &DefaultCollector{
			enabled: false,
		}
	}
}

var (
	Collector EventCollector

	syncerTelemetryEndpoint = "https://admin.loft.sh/analytics/v1/vcluster/v1/syncer"

	// a dummy key so this doesnt fail for dev/testing, this is set by build flag in release action
	telemetryPrivateKey string = `LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlKS1FJQkFBS0NBZ0VBdlE3cHhqYzEybzlJdXQyQkQ2TUtaWnhDY29hbXpJNVV0Wll6Wk5GZFVQYkJsSlI0ClBXOEM1STM3ZHk1cW1yMlU5UlJSbjNlOUpjSDRPS0QzenVHSkhDd0Z2TnpOYzJsYVQ0dlE5NjlVeVpmakdhT3AKVmxtSEhDaDJXajZvbHNUNmhldGJySTNpYzNvVm1XRHBhSHM4OGU3K2dzTnkyTUowNjNES0ZYM0VLV3pNQVVWZQprZUI1M29DWStCT0R0RExRcHd3eC9wQWp1bUFNS0dkNEc2a3FhcE1VZElpN1NKMzlyL2JxL2VUeWZwSUUzOW9ZCmoxanlhdkFpRFMxR1g0Mm5mU0lkck5NSDhERytSSzNVSHMycTFDOXI0Y1dzenVURktlOHprZ2ltdC9oY2sxS2sKZjZBTllyRE0vQmlrcWZoYXVDcHlMdDFhOTdHbUNNb0x1Z3NBWlB6TjhlTXk4eFlwZy9PZ3hjQ3d2a1E2SzVnTQo0Q3k5ZG5aOFVJTlBTVE02Z2xJUFR1cHpPUzYwVDlBb2VZREcwa245bEVYKzhHT1RyZ1NmaTBONEpxbzZ0YlU0Cjk0TXlvcTB2VmtaVjNGYWNKMDcwbFJQaUFxcm1BeDdMS1J0UUNoeiszY0hLcDhBUmNWS1RuT3VpSHdDNEk0SUYKUGhmbzZ4QWFQWUJ0RDRNQUtydFErS3pycVlwNHZaTVZZWUpWR1hzT2xFN1FGSUJjUk5FUTlPdmlTSHZpTEhsTApHbjhER1NxWVVBWEVPVVZBNWtkTUVwOGhYMXJkb2pyNk9FbEt5dnF1Wk9JMk9yekhDWmFCNS9nQVZwelRGcGhYCjlrN2oyU1FyNUcxbXI2TTE3VXJacnhBcUphSGo1ZnBuL2dCOW9ubkN1bmcvSTNDOEVzb0xVQnovd0xrQ0F3RUEKQVFLQ0FnRUFsMGVNcHBCZEpuTks5a1B5VnVuV2t2SVRkWUxyaTNsRXJUendDUGRDM1Z0bUVSY3drNi8xdDU4cApIZmZsVThicG42WlBuZlA1UlhKTnhqcC9zR3BtQlVYd25XeHRkYkZTazU1RWF6MC84a1A0Yy9heXRLYlU1eUkxCmVnYnpiaGxXZ2J5UDBhYURFbllaUEc4QXRoc083R1NhQVZhVjJuN1hnZUh4d25xdGNaeGVMWkl0bHpyeEthcnIKUEc2WkQ2TXR0TTJjWDU5RkI0aDlrZ01oWjdqWWVRa1I4Q0hOQXRGeFF0R292ZHJxYzM4eUtWRmlINnBENkhBWQpQMFVBTDh1d3Z2K0NrVjBYMkFwbHZwejl4Rnc4R3FlTGd0QmpjL1k1RWxJV2lQOGxNTWFxaFRRMjd1ekthVE1pCkE0TlFsN1ZrR2tQVXRFMXAwaE96MFFxamtZM21GSWpsTEhJYTBWN3QybmxrSXJNKzVLbVNqbFF4WjhMRHZGcnoKTkxFaUk5dnp1RWNHdVpPYXZHdjhzSmhXK2paQ3JQWVVyc3dPMTBsYnVsMkdnL2JBVFd2U3lVUGlyb1RsbEc5NApsVzFrMzl1MUk5d0tLaExSaE5TMlZQTW4xSE1CY0FFQXlTTWFudDdwbjQ4R1R1Q0VseUlEZTM4OW9KWHVOcXpNCjdrS2VaaG0wYzBJNmpvSThVcmNyMVZvTTErbmdtdzlxWldtOWJXekZpNW1IaTlCRXkyRGpjeHlOK1l3bFRVQW4Kd0EyblpoMVY0U1hUZUVWUzFOQ3J5dGNXSlZBdjJObWpTV3ZUQk4yaFdCZzEyVGpXZi9MSEF4eklyVzBaa0tKcgptVXdDQ0V3anhkM2JIMzluWnZkeG5xTXVISnZnUUpmM1NGQVRmZlZDWjI2OWdCZUtoZ0VDZ2dFQkFOc2IyTnZ5CmhxU2MwbW43VWtncHYvUjh0TVNsVVRId3g3cDNCZ21xR2tnU0JCR0t3cEI4Q1NzNWl4UHFGYzZaMFlBY0toL0wKZEFGTFNMN3NHRFFweGpleTFmWEtpeXdQUm5MZEZ2aTBac29UcXhsRUJweVBlQjlWSXZLckFuN3cyWVY0VnJIdQpsbGcwV0FmL2s3cTdsMXRGUmxnaFFSN084UVVtS1daUGNGRjhqR1lYZHZhWUdjZzRpSkM4djBBZ1VhQmxPMCt6CnEzaHBvQ0Q5NlRIUUNTdDQvT0E1Rmo2SzUweWk1N2FHd25vMTNBbVQ5eG9Qc0dqeldLMG1oUGp3M0NZQlYwbisKTit4OVBBcXBJdmhPbU1KZmpOWEQzYndoSFNjNUd4VjlpbUpBSk03MjhZazR5Nm9rVTdJNkJXZDNLV3pDaVRENwp5UHE4U1N3amxoaVk3WGtDZ2dFQkFOemp6NlFHZ1V5Zm5PZ1IwR2JEejAyMVQrOUkrdHZwSnVPQmZNWTVsTlo0CndHOGtoSkdFZk5CQUxqOXIzQXcxaDhEbEpSUmp2OVFFR2krYmFpRkJPWVB4VTlyczJnMmhIWkk5ZkdyRW1LUDMKRjJsNE1TNm1vVGx0RUxwZnM5R3FhQmxyRFlYODFYeEVSQTU0aUhQSmRoQjl0K0QxTFdxRXVLMVVaZ25NUlNQLwp2RHdZUEFXOHlvMGhiM0JDMTVvZ3BPRzMyMWl0QmpYVjhtN1pmN25rZjNyOEhWaG5BcytubW93bXg1dU8yanZ6CmtrajdxaC9WRmlHajcxc1ovV0NHQk9pQk0ySVVMNVhQUmpCM2lNQXMzNGtaTHdlOC9SL3NsaE81bzFucVhVeW4KLzV3UTFoYlNKOXpzNzdwYys0K2ROUHdJVzdHSnhEbENkQWZTd0RuNjNVRUNnZ0VBRnFhWFVZMk4yOENXZy94RwpNazJXbVhpMjIwbFh6bmpjdk9zSEJjSysrc3BaLzFJLzhOM1J1TlUzQ25UOWtpRVdwazdERUF4aFRxend0VVFFCjhJZU5CVDhJbldNMTVmVWlURWVNMDJNYTZUTUZVaFJWTnFRaVArTDJQTzN1MFI2bTdnUlZ1Z2szSTZFdHBJNEkKUUpxWitBWitVaWdGNm1Cc1RDTDR6cW5ScTZyYmZNWmFOdjNjVkhWN3NMTENkcWVncUpzdWVYdlNjeDFBUDRqZwpMWlViRFpKeFdlQ3M2d1JEQ3dvZ09COVFSWUFCNGorWW9Pb1VTNVUwaXBuYnp6eGZGZEszcWwrTWVuY3IyTkpKCldqQU4zTEl5QmZzOGxmRTZhVTZlL1NiQVFvM3RBRFJKSGUxd0tJT2UzMkxlSWljUWNqemVIK0Uza3F3YVNHVFoKWkd1U3lRS0NBUUVBbjREeGcyUWZJaEZ2NERSYzVKZ29yZGhyYkVLcXd2bk5WeU05MG5YcUFDVVo4Q2ZTZ3JIRQozeXc1T1JyTnZ4TTRnQlgzZkkyN0M0SWExcDNIT1ZROEVBYkhvcUs5b25IaFJLU1pudzl2bVpibmxRVnhubG84CnVaY0VLVkRLTEhCODB6MzJlZlprd21NWk1jbmYzcHh2WU9FblVvNDR5VjRsYlNRd3VvcUNzc2dNU09qSER1MlEKNWZCcTVBbWdYbStNSUdIL1JqMUs2cjBmWHVRMzB5Z28xY29QOXJJTDJaOFJmbnJTVUlZTEdKZDkzcTI3MzFpagpybzhPWEI2Y1ZJTHlNR0o3bENzM1lWcFhPTkJZTTAwejdXLytBZng2Vy84Zk1BY3c2ZERPcG5mNW45eVllOG90CmR0NnhEVVh2Y1hqM3RiYmpYNFEzNlpFTzhFZEMvNXNqQVFLQ0FRQkJSNHk5OTJyN05LOVZTZUh1TG1VSUZjd2kKS1dnMG0rVGk2TGxyeDFPS2k3b3cydDhubXlDQklNaDhGWTVJL0RsV1JpTTh0WWUrS2VBYTJCUFdpWmRVbUVQUgpKTHpBWVFjNXhYNVNSait3MDNHZjljdzBteFRNazNzbGxSUERYNEZ2bVpSRDkyRHBwWlFBbHdTU3haZHEyWERrCmMrZG9pVE9zTm04endNaFNXVTFiK2d6MjlabnVFamtMeU1HaXVlUll6bXNVdmdjL09KRzdSU2V5NzhmUWtZYm0KWnRIb3o3dFdJTFJxSVRYclFlZ2h6N2Jrc1lPMzM5QjE5bEI0blFPc2Y3MnBMVnMyTWdkRWlwUUp4VDlaVWRmaQpGZkQ1YzNlSkNINlNWQkRSdjFpV3Y5TzRSdWkvNThMcVRvYVE0bzdmRWxha2RoN3BrK3ZiWnRmNG41dlUKLS0tLS1FTkQgUlNBIFBSSVZBVEUgS0VZLS0tLS0K`
)

const (
	eventsCountThreshold = 500
	maxUploadInterval    = 5 * time.Minute
	// minimum time between uploading events
	minUploadInterval = time.Minute
	// initialSyncIgnoreInterval after start we ignore EventResourceSync events
	initialSyncIgnoreInterval = time.Minute
)

type EventCollector interface {
	IsEnabled() bool
	// RecordEvent adds the produced event to a buffer to eventually be sent to the telemetry backend
	RecordEvent(e *types.Event)
	// NewEvent allocates a new Event struct to be populated by the caller.
	NewEvent(t types.EventType) *types.Event
	SetOptions(options *vcontext.VirtualClusterOptions)
	SetVirtualClient(virtualClient *kubernetes.Clientset)
	// start command object is used to determine which flags were set by the user
	SetStartCommand(startCommand *cobra.Command)
}

func NewDefaultCollector(ctx context.Context, config types.SyncerTelemetryConfig) (*DefaultCollector, error) {
	hostConfig, err := ctrl.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get host rest config: %v", err)
	}
	hostClient, err := kubernetes.NewForConfig(hostConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create ClientSet from rest config: %v", err)
	}

	vclusterNamespace, err := clienthelper.CurrentNamespace()
	if err != nil {
		return nil, fmt.Errorf("failed to create ClientSet from rest config: %v", err)
	}

	decodedCertificate, err := base64.RawStdEncoding.DecodeString(telemetryPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode telemetry key string: %v", err)
	}

	privateKey, err := parsePrivateKey(decodedCertificate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse telemetry key: %v", err)
	}

	tokenGenerator, err := serviceaccount.JWTTokenGenerator("vcluster-telemetry", privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWTTokenGenerator: %v", err)
	}

	c := &DefaultCollector{
		config:            config,
		log:               loghelper.New("telemetry"),
		enabled:           true,
		hostClient:        hostClient,
		vclusterNamespace: vclusterNamespace,
		startTime:         time.Now(),

		// events doesn't need to match eventsCountThreshold, we just
		// need to make sure its fast enough emptied.
		events: make(chan *types.Event, 100),
		buffer: newEventBuffer(eventsCountThreshold),

		tokenGenerator: tokenGenerator,
	}

	go c.start()

	return c, nil
}

type DefaultCollector struct {
	config  types.SyncerTelemetryConfig
	log     loghelper.Logger
	enabled bool

	events      chan *types.Event
	buffer      *eventBuffer
	bufferMutex sync.Mutex

	hostClient        *kubernetes.Clientset
	virtualClient     *kubernetes.Clientset
	vclusterNamespace string
	options           *vcontext.VirtualClusterOptions
	startCommand      *cobra.Command

	startTime time.Time
	// lastUploadTime contains the Time of the previous upload
	lastUploadTime time.Time

	tokenGenerator       serviceaccount.TokenGenerator
	token                string
	tokenLastGeneratedAt time.Time
}

func (d *DefaultCollector) IsEnabled() bool {
	return d.enabled
}

func (d *DefaultCollector) NewEvent(t types.EventType) *types.Event {
	return &types.Event{Type: t}
}

func (d *DefaultCollector) RecordEvent(e *types.Event) {
	// ignore initial reconciling events
	if e.Type == types.EventResourceSync && time.Now().Before(d.startTime.Add(initialSyncIgnoreInterval)) {
		return
	}
	e.Time = int(time.Now().UnixMicro())
	d.events <- e
}

func (d *DefaultCollector) SetOptions(options *vcontext.VirtualClusterOptions) {
	d.options = options
}

func (d *DefaultCollector) SetVirtualClient(virtualClient *kubernetes.Clientset) {
	d.virtualClient = virtualClient
}

func (d *DefaultCollector) SetStartCommand(startCommand *cobra.Command) {
	d.startCommand = startCommand
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
		d.executeUpload(context.Background(), events)

		// Exit if the upload was caused by the SIGTERM
		if terminate {
			os.Exit(1)
		}
	}
}

func (d *DefaultCollector) exchangeBuffer() []*types.Event {
	d.bufferMutex.Lock()
	defer d.bufferMutex.Unlock()

	events := d.buffer.Get()
	d.buffer = newEventBuffer(eventsCountThreshold)
	return events
}

// executeUpload assumes that the caller holds the Lock for the uploadMutex
func (d *DefaultCollector) executeUpload(ctx context.Context, buffer []*types.Event) {
	if d.token == "" || d.tokenLastGeneratedAt.Before(time.Now().Add(-time.Hour)) {
		token, err := d.tokenGenerator.GenerateToken(&jwt.Claims{}, &jwt.Claims{})
		if err != nil {
			d.log.Debugf("failed to generate telemetry request signed token: %v", err)

			return
		}

		d.token = token
	}

	r := types.SyncerTelemetryRequest{
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
	r.InstanceProperties = d.getSyncerInstanceProperties(ctx)

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

func (d *DefaultCollector) getSyncerInstanceProperties(ctx context.Context) types.SyncerInstanceProperties {
	p := types.SyncerInstanceProperties{
		UID:                      getSyncerUID(ctx, d.hostClient, d.vclusterNamespace, d.options),
		InstanceCreator:          d.config.InstanceCreator,
		InstanceCreatorUID:       d.config.InstanceCreatorUID,
		Arch:                     runtime.GOARCH,
		OS:                       runtime.GOOS,
		SyncerVersion:            SyncerVersion,
		SyncerFlags:              getSyncerFlags(d.startCommand, d.options),
		VirtualKubernetesVersion: getVirtualKubernetesVersion(d.virtualClient),
		HostKubernetesVersion:    getHostKubernetesVersion(d.hostClient),
		VclusterServiceType:      getVclusterServiceType(ctx, d.hostClient, d.vclusterNamespace, d.options),
	}
	// SyncerPodsReady          int    // TODO: helper function to get syncerPodsReady- not cached
	// SyncerPodsFailing        int    // TODO: helper function to get syncerPodsFailing- not cached
	// SyncerPodCreated         int    // TODO: helper function to get syncerPodCreated- not cached
	// SyncerPodRestarts        int    // TODO: helper function to get syncerPodRestarts- not cached
	// SyncerMemoryRequests     int    // TODO: use (q *Quantity) AsInt64() ? - not cached
	// SyncerMemoryLimits       int    // TODO: use (q *Quantity) AsInt64() ? - not cached
	// SyncerCpuRequests        int    // TODO: use (q *Quantity) AsInt64() ? - not cached
	// SyncerCpuLimits          int    // TODO: use (q *Quantity) AsInt64() ? - not cached

	return p
}
