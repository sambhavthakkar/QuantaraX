#!/bin/bash

# QuantaraX File Transfer Performance Demo
# This script demonstrates real file transfer with speed measurements

set -e

echo "🚀 QuantaraX File Transfer Performance Demo"
echo "=========================================="

# Configuration
TRANSFER_PORT=44330
RECV_DIR="./demo_received"
TEST_FILE="demo_test_file.bin"
LOG_DIR="./transfer_logs"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Cleanup function
cleanup() {
    echo -e "\n${YELLOW}🧹 Cleaning up processes...${NC}"
    pkill -f "quic_recv.*$TRANSFER_PORT" 2>/dev/null || true
    sleep 1
}
trap cleanup EXIT

# Create directories
mkdir -p "$RECV_DIR" "$LOG_DIR"

echo -e "\n${BLUE}📁 Step 1: Creating test file (10MB)${NC}"
dd if=/dev/urandom of="$TEST_FILE" bs=1M count=10 2>/dev/null
FILE_SIZE=$(stat -f%z "$TEST_FILE" 2>/dev/null || stat -c%s "$TEST_FILE")
echo -e "✅ Created test file: $TEST_FILE ($(($FILE_SIZE / 1024 / 1024)) MB)"

echo -e "\n${BLUE}🎯 Step 2: Starting receiver${NC}"
echo "   Listening on port: $TRANSFER_PORT"
echo "   Output directory: $RECV_DIR"

# Start receiver with timing
./bin/quic_recv --listen "localhost:$TRANSFER_PORT" --output-dir "$RECV_DIR" > "$LOG_DIR/receiver.log" 2>&1 &
RECV_PID=$!
sleep 2

echo -e "✅ Receiver started (PID: $RECV_PID)"

echo -e "\n${BLUE}📤 Step 3: Measuring upload performance${NC}"

# Get internet upload speed using speedtest-cli or curl (fallback)
echo "🌐 Checking internet upload speed..."
if command -v speedtest-cli &> /dev/null; then
    UPLOAD_SPEED=$(speedtest-cli --simple | grep "Upload:" | awk '{print $2 " " $3}')
    echo -e "   Internet Upload Speed: ${GREEN}$UPLOAD_SPEED${NC}"
else
    echo "   (speedtest-cli not available, skipping internet speed test)"
    UPLOAD_SPEED="N/A"
fi

echo -e "\n${BLUE}🚀 Step 4: Performing file transfer${NC}"
echo "   File: $TEST_FILE ($(($FILE_SIZE / 1024 / 1024)) MB)"
echo "   Chunk size: 1MB"
echo "   Target: localhost:$TRANSFER_PORT"

# Measure transfer time
START_TIME=$(date +%s.%N)

# Send the file in chunks to simulate real transfer
CHUNK_SIZE=1048576  # 1MB chunks
TOTAL_CHUNKS=$(( ($FILE_SIZE + $CHUNK_SIZE - 1) / $CHUNK_SIZE ))

echo "   Total chunks to send: $TOTAL_CHUNKS"
echo -e "   ${YELLOW}Transfer in progress...${NC}"

for ((i=0; i<$TOTAL_CHUNKS; i++)); do
    OFFSET=$((i * $CHUNK_SIZE))
    printf "\r   Progress: %d/%d chunks (%.1f%%)" $((i+1)) $TOTAL_CHUNKS $(echo "scale=1; ($i+1)*100/$TOTAL_CHUNKS" | bc)
    
    ./bin/quic_send \
        --addr "localhost:$TRANSFER_PORT" \
        --file "$TEST_FILE" \
        --chunk-index "$i" \
        --chunk-size "$CHUNK_SIZE" \
        --offset "$OFFSET" \
        > "$LOG_DIR/sender_chunk_$i.log" 2>&1
        
    # Small delay to prevent overwhelming
    sleep 0.1
done

END_TIME=$(date +%s.%N)
printf "\n"

# Calculate transfer performance
TRANSFER_TIME=$(echo "$END_TIME - $START_TIME" | bc)
TRANSFER_RATE_MBPS=$(echo "scale=2; ($FILE_SIZE / 1024 / 1024) / $TRANSFER_TIME * 8" | bc)
TRANSFER_RATE_MBS=$(echo "scale=2; ($FILE_SIZE / 1024 / 1024) / $TRANSFER_TIME" | bc)

echo -e "\n${GREEN}📊 Transfer Results${NC}"
echo "=========================="
echo "File size: $(($FILE_SIZE / 1024 / 1024)) MB"
echo "Transfer time: ${TRANSFER_TIME} seconds"
echo "Transfer rate: ${TRANSFER_RATE_MBS} MB/s"
echo "Transfer rate: ${TRANSFER_RATE_MBPS} Mbps"
echo "Internet upload speed: $UPLOAD_SPEED"

# Wait for receiver to process
echo -e "\n${BLUE}⏳ Step 5: Verifying received files${NC}"
sleep 2

# Check received chunks
RECEIVED_CHUNKS=$(ls "$RECV_DIR"/chunk_*.bin 2>/dev/null | wc -l)
echo "   Chunks received: $RECEIVED_CHUNKS/$TOTAL_CHUNKS"

if [ "$RECEIVED_CHUNKS" -eq "$TOTAL_CHUNKS" ]; then
    echo -e "   ${GREEN}✅ All chunks received successfully!${NC}"
    
    # Calculate total received file size
    TOTAL_RECEIVED=0
    for chunk in "$RECV_DIR"/chunk_*.bin; do
        if [ -f "$chunk" ]; then
            CHUNK_SIZE_ACTUAL=$(stat -f%z "$chunk" 2>/dev/null || stat -c%s "$chunk")
            TOTAL_RECEIVED=$((TOTAL_RECEIVED + CHUNK_SIZE_ACTUAL))
        fi
    done
    
    echo "   Total received data: $(($TOTAL_RECEIVED / 1024 / 1024)) MB"
    echo "   Data integrity: $((TOTAL_RECEIVED == FILE_SIZE ? 100 : (TOTAL_RECEIVED * 100 / FILE_SIZE)))%"
else
    echo -e "   ${RED}⚠️  Some chunks missing ($RECEIVED_CHUNKS/$TOTAL_CHUNKS)${NC}"
fi

echo -e "\n${BLUE}📋 Step 6: System Information${NC}"
echo "CPU: $(lscpu | grep "Model name" | cut -d: -f2 | xargs)"
echo "Memory: $(free -h | grep "Mem:" | awk '{print $2}')"
echo "Network interface: $(ip route | grep default | awk '{print $5}' | head -1)"

# Show receiver logs
echo -e "\n${BLUE}📝 Receiver Logs (last 10 lines):${NC}"
tail -10 "$LOG_DIR/receiver.log"

echo -e "\n${GREEN}🎉 Transfer demo completed!${NC}"
echo "Log files saved in: $LOG_DIR"
echo "Received files in: $RECV_DIR"

# Cleanup test files
echo -e "\n${YELLOW}🧹 Cleaning up test files...${NC}"
rm -f "$TEST_FILE"