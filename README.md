# go-dynatrace-webhook

Lightweight HTTP service to receive problems from Dynatrace and do something with them

## Handlers

There is one `Handler` implemented, for Zabbix.

It will forward the problem using zabbix_sender to the specified zabbix server in config.json

## Implementing other handlers

Just implement your function, and add the handler to a new route


