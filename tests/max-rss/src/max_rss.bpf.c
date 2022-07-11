#include <linux/bpf.h>
#include <bpf/bpf_helpers.h>

char LICENSE[] SEC("license") = "Dual BSD/GPL";

struct {
	__uint(type, BPF_MAP_TYPE_ARRAY);
	__uint(max_entries, 1);
	__type(key, unsigned int);
	__type(value, long);
} max_rss SEC(".maps");

struct rss_stat_ctx {
	unsigned long unused;
	unsigned int mm_id;
	unsigned int curr;
	int member;
	long size;
};

SEC("tracepoint/kmem/rss_stat")
int handle_tp(struct rss_stat_ctx* ctx)
{
	char comm[16] = {0};
	int err, id;
	long* rss_size;

	if (ctx->member != 1)
		goto out;

	err = bpf_get_current_comm(comm, 16);
	if (err < 0)
		goto out;

	if (!((comm[0] == 'n') && (comm[1] == 'i') && (comm[2] == 'k') && (comm[3] == 'o') && (comm[4] == 's')))
		goto out;

	id = 0;
	rss_size = bpf_map_lookup_elem(&max_rss, &id);
	if (rss_size == NULL)
		goto out;

	if (*rss_size >= ctx->size)
		goto out;

	*rss_size = ctx->size;
out:
	return 0;
}
