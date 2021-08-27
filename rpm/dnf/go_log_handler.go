// +build dnf

package dnf

// #cgo pkg-config: gio-2.0
// #cgo pkg-config: libdnf
// #include <libdnf/libdnf.h>
//
// typedef const gchar cgchar_t;
import "C"

/*
If a Go file contains a function exported for use by C code, then that file's C preamble cannot contain any
definitions, only declarations (see https://golang.org/cmd/cgo/#hdr-C_references_to_Go).

For that reason, go_log_handler could not be declared in dnf.go & had to be placed in it's own file.
*/

//export go_log_handler
func go_log_handler(log_domain *C.cgchar_t, log_level C.GLogLevelFlags, message *C.cgchar_t, data C.gpointer) {
	switch log_level {
	case C.G_LOG_LEVEL_DEBUG:
		logger.Debug(C.GoString(message))
	case C.G_LOG_LEVEL_INFO:
		logger.Info(C.GoString(message))
	case C.G_LOG_LEVEL_WARNING:
		logger.Warn(C.GoString(message))
	case C.G_LOG_LEVEL_ERROR, C.G_LOG_LEVEL_CRITICAL:
		logger.Error(C.GoString(message))
	}
}
