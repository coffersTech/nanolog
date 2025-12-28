#!/bin/bash
set -e

# Default count is 1000 if not provided
# Try to source SDKMAN
export SDKMAN_DIR="$HOME/.sdkman"
[[ -s "$HOME/.sdkman/bin/sdkman-init.sh" ]] && source "$HOME/.sdkman/bin/sdkman-init.sh"

echo "Installing NanoLog Java SDK to local Maven repository..."

# Check Java Version
# Attempt to use the version the user likely installed/is installing
sdk use java 17.0.0-tem || true

if [[ -n "$JAVA_HOME" ]]; then
  echo "Using JAVA_HOME: $JAVA_HOME"
fi
java -version

cd "$(dirname "$0")/../sdks/java/nanolog-spring-boot-starter"

# Extract version from pom.xml
VERSION=$(mvn help:evaluate -Dexpression=project.version -q -DforceStdout)

mvn clean install -DskipTests

echo "✅ SDK Installed Successfully!"
echo "You can now depend on it in your projects:"
echo ""
echo "    <dependency>"
echo "        <groupId>tech.coffers</groupId>"
echo "        <artifactId>nanolog-spring-boot-starter</artifactId>"
echo "        <version>${VERSION}</version>"
echo "    </dependency>"
