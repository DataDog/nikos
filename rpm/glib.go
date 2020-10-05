// +build dnf

package rpm

// #cgo pkg-config: gio-2.0
// #cgo pkg-config: libdnf
// #include <libdnf/libdnf.h>
//
// void gSignalConnect(gpointer obj, gchar *sig, GCallback callback, gpointer data)
// {
//      g_signal_connect(obj, sig, callback, data);
// }
//
// DnfPackage * get_package(GPtrArray *pkglist) {
//     return (DnfPackage*) g_object_ref(g_ptr_array_index(pkglist, 0));
// }
//
// DnfRepo* get_repository(GPtrArray *pkglist, int i) {
//     return (DnfRepo*) g_object_ref(g_ptr_array_index(pkglist, i));
// }
//
// int set_package_repo(DnfContext *ctx, DnfPackage *pkg) {
//     int i;
//     DnfRepo *repo;
//     GPtrArray *repos = dnf_context_get_repos(ctx);
//     for (i = 0; i < repos->len; i++) {
//         repo = (DnfRepo*) g_ptr_array_index(repos, i);
//         if (g_strcmp0(dnf_package_get_reponame(pkg), dnf_repo_get_id(repo)) == 0) {
//             dnf_package_set_repo(pkg, repo);
//             return TRUE;
//         }
//     }
//     return FALSE;
// }
//
// extern void download_percentage_changed_cb(DnfState* dnfState, guint value, gpointer data);
// void dnf_state_set_percentage_changed_cb(DnfState* dnfState) {
//     	g_signal_connect(dnfState, "percentage-changed", G_CALLBACK(download_percentage_changed_cb), NULL);
// }
//
// typedef const gchar cgchar_t;
// extern void go_log_handler(cgchar_t *log_domain, GLogLevelFlags log_level, cgchar_t *message, gpointer data);
// void dnf_set_default_handler() {
//      g_log_set_default_handler(go_log_handler, NULL);
// }
import "C"
import "unsafe"

func g_signal_connect(object C.gpointer, name string, to C.GCallback, data C.gpointer) {
	nameC := (*C.gchar)(unsafe.Pointer(C.CString(name)))
	defer C.free(unsafe.Pointer(nameC))
	C.gSignalConnect(object, nameC, to, data)
}

func getPackage(plist *C.struct__GPtrArray) *C.DnfPackage {
	return C.get_package(plist)
}

func getRepository(plist *C.struct__GPtrArray, i int) *C.DnfRepo {
	return C.get_repository(plist, C.int(i))
}

func set_package_repo(ctx *C.struct__DnfContext, pkg *C.DnfPackage) int {
	return int(C.set_package_repo(ctx, pkg))
}
