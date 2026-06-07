# syntax=docker/dockerfile:1

# Pierwszy etap budowy – wykorzystanie obrazu bazowego golang 1.26.2 Alpine 3.23.
FROM --platform=$BUILDPLATFORM golang:1.26.2-alpine3.23 AS build

# Zainstalowanie certyfikatów SSL (potrzebnych do żądań HTTPS) oraz UPX do kompresji binarki.
RUN apk add --no-cache ca-certificates upx

# Ustawienie katalogu roboczego.
WORKDIR /app

# Skopiowanie pliku z zależnościami przed kodem źródłowym w celu optymalizacji cache.
COPY go.mod .

# Pobranie zależności.
RUN go mod download

# Skopiowanie kodu źródłowego aplikacji i pliku HTML do kontenera (zmiana kodu nie powoduje ponownego pobierania zależności).
COPY main.go index.html .

# Zdefiniowanie zmiennych wskazujących docelowy system operacyjny i docelową architekturę builda (nadpisywane przez docker buildx).
ARG TARGETOS=linux
ARG TARGETARCH=amd64

# Zbudowanie statycznej binarki dla wybranej platformy i próba jej maksymalnego skompresowania.
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -gcflags=all=-l -o weather-raw . && \
	upx --best --lzma -f -o weather weather-raw || cp weather-raw weather

# Drugi etap budowy – wykorzystanie obrazu bazowego scratch.
FROM scratch

# Dodanie metadanych informujących o autorze obrazu.
LABEL org.opencontainers.image.authors="Julia Jurczak <s101581@pollub.edu.pl>"

# Skopiowanie certyfikatów z pierwszego etapu.
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Skopiowanie binarki aplikacji zbudowanej w pierwszym etapie.
COPY --from=build /app/weather /weather

# Wskazanie portu, na którym nasłuchuje kontener.
EXPOSE 8080

# Ustawienie użytkownika na takiego z minimalnymi uprawnieniami (brak /etc/passwd w scratch, 65534:65534 to tradycyjny "nobody:nogroup").
USER 65534:65534

# Zdefiniowanie mechanizmu sprawdzania poprawności działania uruchomionej aplikacji (brak curl/wget w scratch, aplikacja sprawdza się sama flagą "-check").
HEALTHCHECK --interval=10s --timeout=1s \
    CMD /weather -check

# Ustawienie binarki jako domyślny proces kontenera.
ENTRYPOINT ["/weather"]