FROM quay.io/keycloak/keycloak:26.8.2-1 AS builder

ENV KC_DB=postgres
ENV KC_HEALTH_ENABLED=true
ENV KC_METRICS_ENABLED=true
ENV KC_FEATURES=token-exchange,admin-fine-grained-authz

RUN /opt/keycloak/bin/kc.sh build

FROM quay.io/keycloak/keycloak:26.8.2-1

COPY --from=builder /opt/keycloak/ /opt/keycloak/

COPY themes/hotel-key /opt/keycloak/themes/hotel-key

ENV KC_DB=postgres
ENV KC_HEALTH_ENABLED=true
ENV KC_METRICS_ENABLED=true
ENV KC_THEME_STATIC_MAX_AGE=2592000

EXPOSE 8080 8443 9000

ENTRYPOINT ["/opt/keycloak/bin/kc.sh", "start", "--optimized"]
