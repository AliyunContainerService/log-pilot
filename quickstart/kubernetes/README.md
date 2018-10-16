# Integrating with Kubernetes

### First, pre-requisite

- Create a Log project on  Aliyun sls [console](https://sls.console.aliyun.com/#/). You can find the relivent document [here](https://help.aliyun.com/document_detail/48984.html?spm=5176.product28958.6.556.0y3TEz)
- Get your ```Aliyun AccessKeyID``` and ```AccessKeySecret``` from [here](https://ak-console.aliyun.com/#/accesskey)

### Second, Deploy Log-pilot to Kubernetes cluster

You need to clone log-pilot code into your local directory.

Then change your working directory to ```quickstart/kubernetes```. Modify the environment variables to some proper value, thus:

```
[root@iZu kubernetes]# cd quickstart/kubernetes
[root@iZu kubernetes]# export LOG_PROJECT_ID="Your Project ID" ENDPOINT="cn-hangzhou.log.aliyuncs.com" ACCESS_KEY_ID="Your AccessKeyID" ACCESS_KEY_SECRET="Your AccessKeySecret"
[root@iZu kubernetes]# cat log-pilot.yml.tmpl | sed "s@##LOG_PROJECT_ID##@$LOG_PROJECT_ID@g" | sed "s@##ENDPOINT##@$ENDPOINT@g" | sed "s@##ACCESS_KEY_ID##@$ACCESS_KEY_ID@g" | sed "s@##ACCESS_KEY_SECRET##@$ACCESS_KEY_SECRET@g" > log-pilot.yml
[root@iZu kubernetes]# kubectl apply -f log-pilot.yml
[root@iZu kubernetes]# kubectl get po -n kube-system
```

With steps above, You have create a log-pilot DaemonSet successfully.

### Third , Verify your log collector

Run a pod to verify that ```log-pilot``` has collect your log correctly.

In order to let your Pod`s logs being collected, You need to pass an environment variable started with prefix ```aliyun_logs_``` plus logstore name like "logpilot" to the Pod.

Something like below.
```
[root@iZu kubernetes]# kubectl run hello-kube --env "aliyun_logs_logpilot=stdout" --image=registry.cn-hangzhou.aliyuncs.com/google-containers/echoserver:1.4 --port=8080
```

> **Note:**
>
> Log-pilot will automatically created a logstore for you on aliyun if you dont have an logstore pre-created.
>
> This policy is effected only when environment variable ```ALIYUNSLS_NEED_CREATE_LOGSTORE``` is set to ```true```

Now, open your Aliyun sls [console](https://sls.console.aliyun.com/#/) to explore your pod log with great advanced log feature support.

More settings available [here](docs/output/aliyun_sls.md)
