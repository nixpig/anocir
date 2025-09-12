package container

// Lifecycle represents the container lifecycle stages that hooks are
// executed on.
type Lifecycle string

const (
	LifecycleCreateRuntime   Lifecycle = "createRuntime"
	LifecycleCreateContainer Lifecycle = "createContainer"
	LifecycleStartContainer  Lifecycle = "startContainer"
	LifecyclePrestart        Lifecycle = "prestart"
	LifecyclePoststart       Lifecycle = "poststart"
	LifecyclePoststop        Lifecycle = "poststop"
)
