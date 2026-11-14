package v1

// Labels the NetBox sync writes onto every Machine and BareMetalHost it
// imports, derived from the NetBox device record, so callers can select
// imported inventory the way they select any other resource, e.g.
// "kubectl get machines -l netbox.vcluster.com/role=gpu-compute". The sync
// owns and rewrites them all on every reconcile.
const (
	// NetBoxMachineLabelPrefix is the prefix of every label the sync derives
	// from the device record.
	NetBoxMachineLabelPrefix = "netbox.vcluster.com/"

	// NetBoxMachineConnectorLabel is the NetBox connector the machine was read
	// from. Device names are unique within one NetBox and nowhere else, so
	// this is what tells two same-named imports apart.
	NetBoxMachineConnectorLabel = "netbox.vcluster.com/connector"

	// NetBoxMachineSiteLabel is the slug of the NetBox site holding the device.
	NetBoxMachineSiteLabel = "netbox.vcluster.com/site"

	// NetBoxMachineLocationLabel is the slug of the NetBox location holding the device.
	NetBoxMachineLocationLabel = "netbox.vcluster.com/location"

	// NetBoxMachineRackLabel is the name of the NetBox rack holding the device.
	NetBoxMachineRackLabel = "netbox.vcluster.com/rack"

	// NetBoxMachineRoleLabel is the slug of the device's NetBox role, e.g. gpu-compute.
	NetBoxMachineRoleLabel = "netbox.vcluster.com/role"

	// NetBoxMachineDeviceTypeLabel is the slug of the device's NetBox device type.
	NetBoxMachineDeviceTypeLabel = "netbox.vcluster.com/device-type"

	// NetBoxMachineManufacturerLabel is the slug of the device type's manufacturer.
	NetBoxMachineManufacturerLabel = "netbox.vcluster.com/manufacturer"

	// NetBoxMachineStatusLabel is NetBox's own status value, verbatim (active,
	// planned, staged, offline, failed, inventory, decommissioning).
	NetBoxMachineStatusLabel = "netbox.vcluster.com/status"

	// NetBoxMachineAvailabilityLabel is the platform's normalized reading of
	// the device's status and tenancy: Available, Assigned, Unavailable or
	// Unknown.
	NetBoxMachineAvailabilityLabel = "netbox.vcluster.com/availability"

	// NetBoxMachineProvisionableLabel is "true" when the device carries
	// everything a provisioner needs: a BMC address and login, a boot MAC and
	// a serial number.
	NetBoxMachineProvisionableLabel = "netbox.vcluster.com/provisionable"
)

// NetBoxMachineDeviceAnnotation names the NetBox device a platform Machine was
// imported from, as "<connectorRef>.<NetBox device id>". The NetBox sync sets
// it on every Machine it creates and pins the import to the device by it:
// NetBox renames devices in place, so the name is not an identity an import
// can follow, but the connector and the numeric id survive a rename.
const NetBoxMachineDeviceAnnotation = "machines.vcluster.com/netbox-device"

// NetBoxMachineURLAnnotation is the NetBox page of the device a platform
// Machine was imported from, so a machine list can link back to where the
// record is edited. The NetBox sync keeps it current on every Machine it owns.
const NetBoxMachineURLAnnotation = "netbox.vcluster.com/url"

// NetBoxMachineDeviceNameAnnotation is the name of the NetBox device a platform
// Machine was imported from, as NetBox currently calls it. The Machine keeps
// the name it was created with, so this is what follows a rename in NetBox.
const NetBoxMachineDeviceNameAnnotation = "netbox.vcluster.com/device-name"
