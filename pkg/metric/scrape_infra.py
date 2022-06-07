import subprocess
import json
import sys

loader_total_cores = 8

if __name__ == "__main__":
    cmd_get_loader_pct = ['bash', 'scripts/metrics/get_loader_cpu_pct.sh']
    cmd_get_abs_vals = ['bash', 'scripts/metrics/get_node_stats_abs.sh']
    cmd_get_pcts = ['bash', 'scripts/metrics/get_node_stats_percent.sh']

    result = {
        "master_cpu_pct": -99,
        "master_mem_pct": -99,
        "cpu": [],
        "cpu_pct": -99,
        "memory": [],
        "memory_pct": -99,
    }

    loader_cpu_pct, loader_mem_pct = list(
        map(float, subprocess.check_output(cmd_get_loader_pct).decode("utf-8").strip().split())
    )
    loader_cpu_pct /= loader_total_cores #* Normalise to [0, 100]

    abs_out = subprocess.check_output(cmd_get_abs_vals).decode("utf-8")[:-1]
    pcts_out = subprocess.check_output(cmd_get_pcts).decode("utf-8")

    counter = 0
    is_master = True
    for abs_vals, pcts in zip(abs_out.split('\n'), pcts_out.split('\n')):
        if is_master:
            # Record master node.
            result['master_cpu_pct'], result['master_mem_pct'] = list(map(float, pcts[:-1].split('%')))
            result['master_cpu_pct'] = max(0, result['master_cpu_pct'] - loader_cpu_pct)
            result['master_mem_pct'] = max(0, result['master_mem_pct'] - loader_mem_pct)
            is_master = False
            continue

        counter += 1
        cpu, mem = abs_vals.split(',')
        cpu_pct, mem_pct = pcts[:-1].split('%')

        result['cpu'].append(cpu[1:-1])
        result['cpu_pct'] += float(cpu_pct)
        result['memory'].append(mem[1:-1])
        result['memory_pct'] += float(mem_pct)
    
    # Prevent div-0 in the case of single-node.
    if counter != 0:
        result['cpu_pct'] /= counter
        result['memory_pct'] /= counter
    else:
        result['cpu'] = ['']
        result['memory'] = ['']

    print(json.dumps(result))
