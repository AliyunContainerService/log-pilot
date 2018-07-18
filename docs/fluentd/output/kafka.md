### kafka output plugin

#### Environment variables for image `fluentd`

```
- LOGGING_OUTPUT=kafka  # Required, specify your output plugin name
- KAFKA_BROKERS=<broker1_host>:<broker1_port>,<broker2_host>:<broker2_port>...  
- KAFKA_DEFAULT_TOPIC=topic 
```


