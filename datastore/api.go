package datastore

type DriverStatus uint8;
const (
	WaitForDatastore DriverStatus = iota
	ResyncInProgress
	InSync
)

type DriverConfiguration struct {

}

type Driver interface {
	Start()
	// ForceResync()
}

type Callbacks interface {
	OnConfigLoaded()
	OnStatusUpdated(status DriverStatus)
	OnKeyUpdated(key string, value string)
	OnKeyDeleted(key string)
}

type DriverConstructor func(callbacks Callbacks, config *DriverConfiguration) (Driver, error)

func Register(name string, constructor DriverConstructor) {
	// TODO Implement driver registration
}
