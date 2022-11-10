## A note on JFR collection

Because a user would typically run a JFR for a number of minutes we do not wait for this to finish. 

It is advised to use the `--jfr` flag when running the tool to triger a JFR and then run another collection later **without the flag** which should pickup the JFR file. 

JFR files will always appear as 0 bytes on the host machine are are only flushed once the recording is dumped. 

If you request a JFR it will be triggered on _all_ the nodes specified by the coordinator and executor IPs / K8s labels you specify. If for example you want to only run a JFR on the coordinator pod for a k8s install you could run:

`... -c default:app=dremio-coodinator -e default:dummy-label ...`

We only trigger a JFR if there is not already one running. While it is possible to run multiple JFRs against a JVM it was decided not to allow this since a user might unintentionally trigger many JFRs which could lead to unpredictable behaviour. 

For more information on Flight Recorder please refer to the oracle documentation

https://docs.oracle.com/javacomponents/jmc-5-4/jfr-runtime-guide/about.htm

