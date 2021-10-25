# [fluent-bit](https://fluentbit.io/) integrated in erda

## 潜在问题
 - Offset_Key 在多行的时候会丢失，现阶段暂时不取offset，以纳秒时间戳应该足够排序了