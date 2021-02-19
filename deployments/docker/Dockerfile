# SPDX-FileCopyrightText: © 2020 Grégoire Duchêne <gduchene@awhk.org>
# SPDX-License-Identifier: ISC

FROM golang:1.15
WORKDIR /root
COPY . ./
RUN go build ./cmd/fwdsms

FROM scratch
COPY --from=0 /root/fwdsms /fwdsms

EXPOSE 8080
ENTRYPOINT ["/fwdsms"]
