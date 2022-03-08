# [fluent-bit](https://fluentbit.io/) integrated in erda

## 字段名规定
- `__tags_*`：日志标签，最终通过`nest`生成tags
```json
{
  "tags":{
    "level": "INFO"
  }
}
```
- `__labels_*`：日志导出配置的标签，最终通过`nest`生成labels，用以兼容日志分析&日志导出组件
```json
{
  "labels":{
    "monitor_log_output": "elasticsearch"
  }
}
```
- `__pri_*`: 私有的中间Key，最终会被删除

## 日志模拟
### dd
### flog
1. 安装：`go install github.com/mingrammer/flog@latest`
2. 生成：
`flog -t log -o testdata/k8s/test.log -b 1073741824 -s 10ms -w`
生成1G大小日志`testdata/k8s/test.log`，