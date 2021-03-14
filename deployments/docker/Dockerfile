# SPDX-FileCopyrightText: © 2020 Grégoire Duchêne <gduchene@awhk.org>
# SPDX-License-Identifier: ISC

FROM golang:1.16
WORKDIR /root
COPY . ./
RUN CGO_ENABLED=0 go build ./cmd/fwdsms

FROM scratch
COPY --from=0 /root/fwdsms /usr/bin/fwdsms

EXPOSE 8080
ENTRYPOINT ["/usr/bin/fwdsms"]
