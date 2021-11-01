# [fluent-bit](https://fluentbit.io/) integrated in erda

## 潜在问题
 - Offset_Key 在多行的时候会丢失，现阶段暂时不取offset，以纳秒时间戳应该足够排序了

## 性能
经测试，现有配置下，1C的输出流量能达到600KB/s-1000KB/s之间, 换算下来，可保证大致3.2MB/s的日志写入而不延迟。

## TODO 优化
- 支持并行json.marshal&compress
- 替换std gzip