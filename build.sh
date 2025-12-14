#!/bin/bash

set -e

echo "Cleaning up previous build..."
rm -rf dist

echo "Building web frontend..."
cd web || exit
npm install
npm run build
cd ..

mkdir -p dist/web_dist
cp -r web/dist/* dist/web_dist/

echo "Building Go backend..."
echo "Building server..."
go build -v -o dist/mc-manager-server ./cmd/server
echo "Building CLI..."
go build -v -o dist/mc-cli ./cmd/cli

echo "Build finished successfully!"
