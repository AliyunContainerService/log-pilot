# Helm

**注意：**

- 使用Helm v3
- 请先安装helm
- helm默认使用`~/.kube/config`集群配置文件
- 使用filebeat，还是fluentd镜像，请修改镜像。
- 输出到kafka，还是es，请修改对应变量配置。其它输出对象，请参考此两者的文件，自己写。



<br>
<br>


## 安装

```sh
cd helm

# 调试
helm template --debug log-pilot log-pilot/

# 默认输出到kafka，如果需要修改kafka配置，请修改对应文件
# 安装
helm install log-pilot log-pilot/


# 也可指定对象
helm --namespace="default" install --set image.tag="0.9.7-filebeat" -f log-pilot/toKafka.yaml log-pilot log-pilot/

# 查看
helm -n default ls
```
