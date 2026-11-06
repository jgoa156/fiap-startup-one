# Front Next em modo autônomo (output: "standalone").
# Contexto de construção: fiap-startup-one/poc-web
FROM public.ecr.aws/docker/library/node:24-alpine AS construcao
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

FROM public.ecr.aws/docker/library/node:24-alpine
WORKDIR /app
ENV NODE_ENV=production NEXT_TELEMETRY_DISABLED=1 PORT=8080 HOSTNAME=0.0.0.0
COPY --from=construcao /app/.next/standalone ./
COPY --from=construcao /app/.next/static ./.next/static
COPY --from=construcao /app/public ./public
USER node
EXPOSE 8080
CMD ["node", "server.js"]
