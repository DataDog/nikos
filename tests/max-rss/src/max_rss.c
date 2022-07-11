#include <stdio.h>
#include <unistd.h>
#include <sys/resource.h>
#include <bpf/libbpf.h>
#include <bpf/bpf.h>
#include <fcntl.h>
#include <signal.h>
#include "max_rss.skel.h"

#define PID_FILE "/tmp/max_rss.pid"
#define LOG_FILE "/tmp/max_rss.log"

typedef void (*sighandler_t)(int);

static int libbpf_print_fn(enum libbpf_print_level level, const char *format, va_list args)
{
	return vfprintf(stderr, format, args);
}

static int write_pid_file() {
	int fd, err;
	char pid_str[5] = {0};
	pid_t pid = getpid();

	sprintf(pid_str, "%d", pid);

	fd = open(PID_FILE, O_WRONLY | O_CREAT);
	if (fd < 0)
		return fd;

	err = write(fd, pid_str, 5);
	if (err < 0)
		return err;

	return close(fd);
}

static void handler(int sig) {
	return;	
}

static int setup_signal(int sig, sighandler_t handler) {
	struct sigaction new;

	new.sa_handler = handler;
	sigemptyset(&new.sa_mask);
	
	new.sa_flags = SA_RESTART;

	return sigaction(sig, &new, NULL);
}

int main(int argc, char **argv)
{
	struct max_rss_bpf *skel;
	struct bpf_object *obj;
	int err, id, fd;
	char max_str[10] = {0};
	long max;

	err = write_pid_file();
	if (err < 0) {
		fprintf(stderr, "Failed to write pid file\n");
		return -1;
	}

	err = setup_signal(SIGINT, handler);
	if (err < 0) {
		fprintf(stderr, "Failed to setup signal handler\n");
		return -1;
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

	pause();

	obj = skel->obj;
	fd = bpf_object__find_map_fd_by_name(obj, "max_rss");
	if (fd < 0)
		goto cleanup;

	id = 0;
	err = bpf_map_lookup_elem(fd, &id, &max);
	if (err < 0)
		goto cleanup;


	sprintf(max_str, "%ld\n", max);
	fd = open(LOG_FILE, O_WRONLY | O_CREAT);
	if (fd < 0)
		goto cleanup;

	err = write(fd, max_str, 9);
	if (err < 0)
		goto cleanup;

	close(fd);

cleanup:
	max_rss_bpf__destroy(skel);
out:
	return -err;
}