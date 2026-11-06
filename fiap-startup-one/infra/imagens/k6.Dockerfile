# Gerador de carga: o k6 com os roteiros do ensaio. Roda como tarefa do ECS, dentro da AWS, para
# que o limite medido seja o do sistema e não o da internet de quem dispara.
# Contexto de construção: fiap-startup-one/carga
#
# O binário vem do release oficial, conferido pelo checksum publicado (e não do Docker Hub, que
# limita downloads anônimos vindos dos IPs compartilhados do CodeBuild).
FROM public.ecr.aws/docker/library/alpine:3.22 AS baixar
ARG K6=v2.3.0
RUN apk add --no-cache curl \
 && cd /tmp \
 && curl -fsSLO "https://github.com/grafana/k6/releases/download/${K6}/k6-${K6}-linux-amd64.tar.gz" \
 && curl -fsSLO "https://github.com/grafana/k6/releases/download/${K6}/k6-${K6}-checksums.txt" \
 && grep " k6-${K6}-linux-amd64.tar.gz$" "k6-${K6}-checksums.txt" | sha256sum -c - \
 && tar -xzf "k6-${K6}-linux-amd64.tar.gz" \
 && mv "k6-${K6}-linux-amd64/k6" /k6

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=baixar /k6 /k6
COPY . /carga
WORKDIR /carga
ENTRYPOINT ["/k6"]
CMD ["run", "ultimo-dia.js"]
