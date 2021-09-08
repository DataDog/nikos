// +build dnf
// +build molecule

package dnf

// #cgo LDFLAGS: -Wl,--wrap=log -Wl,--wrap=pow
// #include "glib_log_pow_wrapper.h"
import "C"