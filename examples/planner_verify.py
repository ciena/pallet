#!/usr/bin/env python3

import os
import sys
import subprocess
import json
import argparse
from argparse import ArgumentParser

def verify(namespace):
    p = subprocess.run("kubectl get pods -n {} -o json".format(namespace),
                       check=True, shell=True, stdout=subprocess.PIPE, universal_newlines=True)
    data = json.loads(p.stdout)
    assignment_map = {}
    for item in data['items']:
        pod, node = item['metadata']['name'], item['spec']['nodeName']
        assignment_map[pod] = node

    p = subprocess.run("kubectl get scheduleplans -n {} -o json".format(namespace),
                       check=True, shell=True, stdout=subprocess.PIPE, universal_newlines=True)
    data = json.loads(p.stdout)
    planner_map = {}

    if len(data['items']) == 0:
        print('No scheduleplan instances found')
        return []
    
    for item in data['items']:
        for p in item['spec']['plan']:
            planner_map[p['pod']] = p['node']
        break

    mismatch = []
    for pod, node in planner_map.items():
        # check if it exists in the planner and matches
        if pod in assignment_map and assignment_map[pod] == node:
            continue
        mismatch.append({'pod':pod,'node': node})

    return mismatch

def main(args):
    mismatch = verify(args.namespace)
    if len(mismatch) != 0:
        print('Planner assignment verification failed. Mismatches', mismatch)
    else:
        print('Planner assignment verification success')

    return 0 if len(mismatch) == 0 else 1

if __name__ == '__main__':
    parser = ArgumentParser(description='Schedule planner assignment verifier',
                            formatter_class=argparse.ArgumentDefaultsHelpFormatter)
    parser.add_argument('-n', '--namespace', default='default', type=str, help='Specify namespace to verify against.')
    args = parser.parse_args()
    sys.exit(main(args))
