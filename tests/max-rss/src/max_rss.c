#include <stdio.h>
#include <unistd.h>
#include <stdlib.h>
#include <sys/resource.h>
#include <bpf/libbpf.h>
#include <bpf/bpf.h>
#include <fcntl.h>
#include "max_rss.skel.h"

static int libbpf_print_fn(enum libbpf_print_level level, const char *format, va_list args)
{
	return vfprintf(stderr, format, args);
}

int main(int argc, char **argv)
{
	struct max_rss_bpf *skel;
	struct bpf_object *obj;
	int err, id, fd;
	char max_str[10] = {0};
	long max;

    if (argc <= 1) {
        fprintf(stderr, "not enough arguments passed to 'max_rss'\n");
        exit(1);
    }
    if ((argv+sizeof(char *)) == NULL) {
        fprintf(stderr, "invalid argument passed to 'max_rss'\n");
        exit(1);
    }

	libbpf_set_strict_mode(LIBBPF_STRICT_ALL);
	/* Set up libbpf errors and debug info callback */
	libbpf_set_print(libbpf_print_fn);

	/* Open BPF application */
	skel = max_rss_bpf__open();
	if (!skel) {
		fprintf(stderr, "Failed to open BPF skeleton\n");
		return 1;
	}

	/* Load & verify BPF programs */
	err = max_rss_bpf__load(skel);
	if (err) {
		fprintf(stderr, "Failed to load and verify BPF skeleton\n");
		goto cleanup;
	}

	/* Attach tracepoint handler */
	err = max_rss_bpf__attach(skel);
	if (err) {
		fprintf(stderr, "Failed to attach BPF skeleton\n");
		goto cleanup;
	}

    printf("%s\n", argv[1]);
    system(argv[1]);
    
	obj = skel->obj;
	fd = bpf_object__find_map_fd_by_name(obj, "max_rss");
	if (fd < 0)
		goto cleanup;

	id = 0;
	err = bpf_map_lookup_elem(fd, &id, &max);
	if (err < 0)
		goto cleanup;

    printf("%lu\n", max);

cleanup:
	max_rss_bpf__destroy(skel);
out:
	return -err;
}
