# SPDX-FileCopyrightText: © 2020 Grégoire Duchêne <gduchene@awhk.org>
# SPDX-License-Identifier: ISC

FROM golang:1.15
WORKDIR /root
COPY . ./
RUN go install ...

FROM scratch
COPY --from=0 /go/bin/fwdsms /usr/bin/fwdsms

EXPOSE 8008
ENTRYPOINT ["/usr/bin/fwdsms"]
