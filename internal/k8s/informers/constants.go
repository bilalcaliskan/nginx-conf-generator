package informers

const (
	ErrRenderTemplate = "an error occurred while rendering template"
	ErrReloadNginx    = "an error occurred while reloading Nginx service"
	ErrApplyChanges   = "fatal error occured while applying changes"
	WarnWorkerLength  = "length of cluster.Workers is 0, can not add a server without any upstream server"
)
