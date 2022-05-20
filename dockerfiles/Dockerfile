FROM registry.erda.cloud/retag/golang:1.16.6 as gobuilder

COPY . /build
WORKDIR /build

ENV GOPROXY="https://goproxy.cn"

RUN make build

FROM registry.erda.cloud/retag/fluent-bit:1.9.3-debug

COPY conf/ /fluent-bit/etc/
COPY entrypoint.sh /fluent-bit/etc/
COPY --from=gobuilder /build/out_erda.so /fluent-bit/bin/
RUN ldd /fluent-bit/bin/out_erda.so

# Entry point
CMD ["/fluent-bit/etc/entrypoint.sh"]