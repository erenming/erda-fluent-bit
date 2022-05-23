FROM registry.erda.cloud/retag/golang:1.16.6 as gobuilder

COPY . /build
WORKDIR /build

ENV GOPROXY="https://goproxy.cn"

RUN make build

FROM registry.erda.cloud/retag/debian:buster-slim as builder

# Fluent Bit version
ENV FLB_MAJOR 1
ENV FLB_MINOR 9
ENV FLB_PATCH 3
ENV FLB_VERSION 1.9.3

#ARG FLB_TARBALL=https://github.com/fluent/fluent-bit/archive/v$FLB_VERSION.tar.gz
ARG FLB_TARBALL=https://erda-project.oss-cn-hangzhou.aliyuncs.com/fluent-bit/fluent-bit-$FLB_VERSION.tar.gz
ENV FLB_SOURCE $FLB_TARBALL
RUN mkdir -p /fluent-bit/bin /fluent-bit/etc /fluent-bit/log /tmp/fluent-bit-master/

ENV DEBIAN_FRONTEND noninteractive

# use tsinghua mirror
RUN mv /etc/apt/sources.list /tmp/
RUN echo "deb http://mirrors.tuna.tsinghua.edu.cn/debian/ buster main contrib non-free" > /etc/apt/sources.list
RUN echo "deb http://mirrors.tuna.tsinghua.edu.cn/debian/ buster-updates main contrib non-free" >> /etc/apt/sources.list
RUN echo "deb http://mirrors.tuna.tsinghua.edu.cn/debian/ buster-backports main contrib non-free" >> /etc/apt/sources.list
RUN echo "deb http://mirrors.tuna.tsinghua.edu.cn/debian-security buster/updates main contrib non-free" >> /etc/apt/sources.list
RUN true

COPY filter_throttle_patch /tmp/filter_throttle_patch
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    build-essential \
    curl \
    ca-certificates \
    cmake \
    make \
    tar \
    libssl-dev \
    libsasl2-dev \
    pkg-config \
    libsystemd-dev \
    zlib1g-dev \
    libpq-dev \
    postgresql-server-dev-all \
    flex \
    bison \
    && curl -L -o "/tmp/fluent-bit.tar.gz" ${FLB_SOURCE} \
    && cd tmp/ && mkdir fluent-bit \
    && tar zxfv fluent-bit.tar.gz -C ./fluent-bit --strip-components=1 \
    && cd fluent-bit/build/ \
    && rm -rf /tmp/fluent-bit/build/* \
    && if [ "$FLB_VERSION" = "1.9.3" ] ; then rm -rf /tmp/fluent-bit/plugins/filter_throttle/throttle.c;  rm -rf /tmp/fluent-bit/plugins/filter_throttle/throttle.h; cp /tmp/filter_throttle_patch/v1.9.3/* /tmp/fluent-bit/plugins/filter_throttle/; fi

WORKDIR /tmp/fluent-bit/build/
RUN cmake -DFLB_RELEASE=On \
          -DFLB_TRACE=Off \
          -DFLB_JEMALLOC=On \
          -DFLB_TLS=On \
          -DFLB_SHARED_LIB=Off \
          -DFLB_EXAMPLES=Off \
          -DFLB_HTTP_SERVER=On \
          -DFLB_IN_SYSTEMD=On \
          -DFLB_OUT_KAFKA=On \
          -DFLB_OUT_PGSQL=On ..

RUN make -j $(getconf _NPROCESSORS_ONLN)
RUN install bin/fluent-bit /fluent-bit/bin/

# Configuration files
#COPY conf/fluent-bit.conf \
#     conf/parsers.conf \
#     conf/parsers_ambassador.conf \
#     conf/parsers_java.conf \
#     conf/parsers_extra.conf \
#     conf/parsers_openstack.conf \
#     conf/parsers_cinder.conf \
#     conf/plugins.conf \
#     /fluent-bit/etc/

COPY conf/ /fluent-bit/etc/
COPY entrypoint.sh /fluent-bit/etc/

COPY --from=gobuilder /build/out_erda.so /fluent-bit/bin/
RUN ldd /fluent-bit/bin/out_erda.so

COPY --from=registry.erda.cloud/retag/busybox:1.31.1 /bin/busybox /bin/busybox

RUN chmod 555 /bin/busybox && \
    /bin/busybox --install -s /usr/local/bin

FROM registry.erda.cloud/retag/debian:buster-slim
MAINTAINER Eduardo Silva <eduardo@treasure-data.com>
LABEL Description="Fluent Bit docker image for testing branch master" Vendor="Fluent Organization" Version="1.1"
RUN true

# use tsinghua mirror
RUN mv /etc/apt/sources.list /tmp/
RUN echo "deb http://mirrors.tuna.tsinghua.edu.cn/debian/ buster main contrib non-free" > /etc/apt/sources.list
RUN echo "deb http://mirrors.tuna.tsinghua.edu.cn/debian/ buster-updates main contrib non-free" >> /etc/apt/sources.list
RUN echo "deb http://mirrors.tuna.tsinghua.edu.cn/debian/ buster-backports main contrib non-free" >> /etc/apt/sources.list
RUN echo "deb http://mirrors.tuna.tsinghua.edu.cn/debian-security buster/updates main contrib non-free" >> /etc/apt/sources.list
RUN true

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    bash gdb valgrind build-essential vim curl sqlite3

# Copy certificates
COPY --from=builder /usr/share/ca-certificates/  /usr/share/ca-certificates/
COPY --from=builder /etc/ssl/ /etc/ssl/

# SSL stuff
COPY --from=builder /usr/lib/x86_64-linux-gnu/*sasl* /usr/lib/x86_64-linux-gnu/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libz* /usr/lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/libz* /lib/x86_64-linux-gnu/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libssl.so* /usr/lib/x86_64-linux-gnu/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libcrypto.so* /usr/lib/x86_64-linux-gnu/
# These below are all needed for systemd
COPY --from=builder /lib/x86_64-linux-gnu/libsystemd* /lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/libselinux.so* /lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/liblzma.so* /lib/x86_64-linux-gnu/
COPY --from=builder /usr/lib/x86_64-linux-gnu/liblz4.so* /usr/lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/libgcrypt.so* /lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/libpcre.so* /lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/libgpg-error.so* /lib/x86_64-linux-gnu/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libpq.so* /usr/lib/x86_64-linux-gnu/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libgssapi* /usr/lib/x86_64-linux-gnu/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libldap* /usr/lib/x86_64-linux-gnu/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libkrb* /usr/lib/x86_64-linux-gnu/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libk5crypto* /usr/lib/x86_64-linux-gnu/
COPY --from=builder /usr/lib/x86_64-linux-gnu/liblber* /usr/lib/x86_64-linux-gnu/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libgnutls* /usr/lib/x86_64-linux-gnu/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libp11-kit* /usr/lib/x86_64-linux-gnu/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libidn2* /usr/lib/x86_64-linux-gnu/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libunistring* /usr/lib/x86_64-linux-gnu/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libtasn1* /usr/lib/x86_64-linux-gnu/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libnettle* /usr/lib/x86_64-linux-gnu/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libhogweed* /usr/lib/x86_64-linux-gnu/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libgmp* /usr/lib/x86_64-linux-gnu/
COPY --from=builder /usr/lib/x86_64-linux-gnu/libffi* /usr/lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/libcom_err* /lib/x86_64-linux-gnu/
COPY --from=builder /lib/x86_64-linux-gnu/libkeyutils* /lib/x86_64-linux-gnu/

COPY --from=builder /fluent-bit /fluent-bit

COPY --from=builder /usr/local/bin/ /usr/local/bin/
COPY --from=builder /bin/busybox /bin/

# Entry point
CMD ["/fluent-bit/etc/entrypoint.sh"]