#!/usr/bin/env python3

from consul_kv import Connection
import argparse
import json
import yaml
import sys

path_to_parse = sys.argv[1]

#Consul URL for STG2: http://10.3.4.6:8500/ui/csstg/kv/test/

# Set Connection to Consul
c = Connection(endpoint='http://10.3.4.6:8500/v1/')

# Get Consul KV as data Dictionary
data = c.get_dict(path_to_parse)

# Store tenant config as yaml
with open("tenant_config_test.yaml", "w") as yml:
    yaml.dump(
    data,
    yml,
    default_flow_style=False)
