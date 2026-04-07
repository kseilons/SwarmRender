#!/bin/bash

# Configuration
KC_ADDR=${KEYCLOAK_ADDR:-"http://keycloak:8080"}
REALM_NAME=${TARGET_REALM:-"render-farm"}
CLIENT_ID=${TARGET_CLIENT_ID:-"auth-service"}
CLIENT_SECRET=${TARGET_CLIENT_SECRET:-"secret"}

# Extract host and port for the wait loop
# Removes http:// or https:// and then everything after the first colon or slash
KC_HOST=$(echo $KC_ADDR | sed -e 's|^[^/]*//||' -e 's|[:/].*||')
KC_PORT=$(echo $KC_ADDR | sed -e 's|^.*:||' -e 's|/.*||')
KC_PORT=${KC_PORT:-8080}

echo "Waiting for Keycloak to be ready at $KC_HOST:$KC_PORT..."
until bash -c "exec 3<>/dev/tcp/$KC_HOST/$KC_PORT" 2>/dev/null; do
  sleep 2
done
echo "Keycloak is up!"

# Login
/opt/keycloak/bin/kcadm.sh config credentials --server "$KC_ADDR" --realm master --user "$KEYCLOAK_ADMIN" --password "$KEYCLOAK_ADMIN_PASSWORD"

# Check if realm exists
if /opt/keycloak/bin/kcadm.sh get "realms/$REALM_NAME" > /dev/null 2>&1; then
  echo "Realm $REALM_NAME already exists."
else
  echo "Creating realm $REALM_NAME..."
  /opt/keycloak/bin/kcadm.sh create realms -s realm="$REALM_NAME" -s enabled=true
fi

# Check if confidential client exists (auth-service)
CLIENT_UUID=$(/opt/keycloak/bin/kcadm.sh get clients -r "$REALM_NAME" --query clientId="$CLIENT_ID" --fields id -format csv --noquotes)

if [ -n "$CLIENT_UUID" ]; then
  echo "Client $CLIENT_ID already exists (ID: $CLIENT_UUID)."
else
  echo "Creating client $CLIENT_ID..."
  # Create client as confidential with a fixed secret
  /opt/keycloak/bin/kcadm.sh create clients -r "$REALM_NAME" \
    -s clientId="$CLIENT_ID" \
    -s enabled=true \
    -s secret="$CLIENT_SECRET" \
    -s serviceAccountsEnabled=true \
    -s publicClient=false \
    -s "redirectUris=[\"*\"]"
fi

# Check if public client exists (frontend)
FE_CLIENT_ID="render-farm-frontend"
FE_CLIENT_UUID=$(/opt/keycloak/bin/kcadm.sh get clients -r "$REALM_NAME" --query clientId="$FE_CLIENT_ID" --fields id -format csv --noquotes)

if [ -n "$FE_CLIENT_UUID" ]; then
  echo "Client $FE_CLIENT_ID already exists (ID: $FE_CLIENT_UUID)."
else
  echo "Creating client $FE_CLIENT_ID..."
  # Create public client for frontend (SPA)
  /opt/keycloak/bin/kcadm.sh create clients -r "$REALM_NAME" \
    -s clientId="$FE_CLIENT_ID" \
    -s enabled=true \
    -s publicClient=true \
    -s "redirectUris=[\"http://localhost:5173/*\"]" \
    -s "webOrigins=[\"http://localhost:5173\"]" \
    -s "attributes={\"pkce.code.challenge.method\":\"S256\"}"
fi

# Create a test user if it doesn't exist
TEST_USER="testuser"
if /opt/keycloak/bin/kcadm.sh get "users" -r "$REALM_NAME" --query username="$TEST_USER" | grep "$TEST_USER" > /dev/null; then
  echo "Test user $TEST_USER already exists."
else
  echo "Creating test user $TEST_USER..."
  /opt/keycloak/bin/kcadm.sh create users -r "$REALM_NAME" -s username="$TEST_USER" -s enabled=true -s emailVerified=true
  /opt/keycloak/bin/kcadm.sh set-password -r "$REALM_NAME" --username "$TEST_USER" --new-password "testpass"
fi

echo "Keycloak setup completed successfully!"
