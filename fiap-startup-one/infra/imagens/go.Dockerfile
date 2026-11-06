# Uma imagem com os quatro binários Go; a tarefa do ECS escolhe qual roda pelo comando.
# Contexto de construção: fiap-startup-one/poc
FROM public.ecr.aws/docker/library/golang:1.27 AS construcao
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -trimpath -ldflags="-s -w" -o /saida/lisura ./cmd/lisura \
 && go build -trimpath -ldflags="-s -w" -o /saida/psp-simulado ./cmd/psp-simulado \
 && go build -trimpath -ldflags="-s -w" -o /saida/lisura-seed ./cmd/seed \
 && go build -trimpath -ldflags="-s -w" -o /saida/lisura-auditoria ./cmd/auditoria

# distroless: sem shell e sem gerenciador de pacotes; roda como usuário sem privilégio.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=construcao /saida/ /
USER nonroot:nonroot
# Sem ENTRYPOINT: o comando escolhe o binário, e uma tarefa avulsa (seed, auditoria) troca só ele.
CMD ["/lisura"]
