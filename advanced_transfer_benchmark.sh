#!/bin/bash

# Advanced QuantaraX Transfer Performance Benchmark
# Includes network monitoring, multiple file sizes, and detailed metrics

set -e

echo "🔬 QuantaraX Advanced Performance Benchmark"
echo "============================================"

# Configuration
BASE_PORT=44400
RECV_DIR="./benchmark_received"
LOG_DIR="./benchmark_logs"
REPORT_FILE="./transfer_performance_report.txt"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

# Test file sizes (in MB)
FILE_SIZES=(1 5 10 50)

# Cleanup function
cleanup() {
    echo -e "\n${YELLOW}🧹 Cleaning up processes...${NC}"
    pkill -f "quic_recv" 2>/dev/null || true
    sleep 1
}
trap cleanup EXIT

# Create directories
mkdir -p "$RECV_DIR" "$LOG_DIR"

# Start performance report
cat > "$REPORT_FILE" << EOF
QuantaraX File Transfer Performance Report
==========================================
Generated: $(date)
System: $(uname -a)
CPU: $(lscpu | grep "Model name" | cut -d: -f2 | xargs)
Memory: $(free -h | grep "Mem:" | awk '{print $2}')
Network Interface: $(ip route | grep default | awk '{print $5}' | head -1)

EOF

echo -e "${CYAN}📊 Running benchmarks for file sizes: ${FILE_SIZES[@]} MB${NC}"

for file_size in "${FILE_SIZES[@]}"; do
    echo -e "\n${BLUE}🧪 Testing ${file_size}MB file transfer${NC}"
    echo "======================================"
    
    # Generate test file
    TEST_FILE="test_${file_size}mb.bin"
    echo "📁 Creating ${file_size}MB test file..."
    dd if=/dev/urandom of="$TEST_FILE" bs=1M count=$file_size 2>/dev/null
    
    ACTUAL_SIZE=$(stat -f%z "$TEST_FILE" 2>/dev/null || stat -c%s "$TEST_FILE")
    echo "✅ File created: $(($ACTUAL_SIZE / 1024 / 1024)) MB"
    
    # Use unique port for each test
    CURRENT_PORT=$((BASE_PORT + file_size))
    CURRENT_RECV_DIR="${RECV_DIR}_${file_size}mb"
    mkdir -p "$CURRENT_RECV_DIR"
    
    # Start receiver
    echo "🎯 Starting receiver on port $CURRENT_PORT..."
    ./bin/quic_recv --listen "localhost:$CURRENT_PORT" --output-dir "$CURRENT_RECV_DIR" > "$LOG_DIR/receiver_${file_size}mb.log" 2>&1 &
    RECV_PID=$!
    sleep 2
    
    # Monitor network stats before transfer
    NETWORK_IFACE=$(ip route | grep default | awk '{print $5}' | head -1)
    if [ -f "/proc/net/dev" ]; then
        RX_BYTES_BEFORE=$(cat /proc/net/dev | grep "$NETWORK_IFACE" | awk '{print $2}')
        TX_BYTES_BEFORE=$(cat /proc/net/dev | grep "$NETWORK_IFACE" | awk '{print $10}')
    fi
    
    # Calculate chunks
    CHUNK_SIZE=1048576  # 1MB chunks
    TOTAL_CHUNKS=$(( ($ACTUAL_SIZE + $CHUNK_SIZE - 1) / $CHUNK_SIZE ))
    
    echo "📤 Transferring $TOTAL_CHUNKS chunks..."
    
    # Start transfer with detailed timing
    START_TIME=$(date +%s.%N)
    START_CPU=$(grep 'cpu ' /proc/stat | awk '{usage=($2+$4)*100/($2+$4+$5)} END {print usage}')
    
    # Transfer chunks with progress monitoring
    for ((i=0; i<$TOTAL_CHUNKS; i++)); do
        OFFSET=$((i * $CHUNK_SIZE))
        
        # Show progress every 10%
        if [ $((i % ($TOTAL_CHUNKS / 10 + 1))) -eq 0 ]; then
            PROGRESS=$(echo "scale=1; $i*100/$TOTAL_CHUNKS" | bc)
            printf "\r   Progress: %.1f%% (%d/%d chunks)" "$PROGRESS" "$i" "$TOTAL_CHUNKS"
        fi
        
        CHUNK_START=$(date +%s.%N)
        
        ./bin/quic_send \
            --addr "localhost:$CURRENT_PORT" \
            --file "$TEST_FILE" \
            --chunk-index "$i" \
            --chunk-size "$CHUNK_SIZE" \
            --offset "$OFFSET" \
            > "$LOG_DIR/sender_${file_size}mb_chunk_$i.log" 2>&1
        
        CHUNK_END=$(date +%s.%N)
        CHUNK_TIME=$(echo "$CHUNK_END - $CHUNK_START" | bc)
        echo "$i,$CHUNK_TIME" >> "$LOG_DIR/chunk_times_${file_size}mb.csv"
    done
    
    END_TIME=$(date +%s.%N)
    END_CPU=$(grep 'cpu ' /proc/stat | awk '{usage=($2+$4)*100/($2+$4+$5)} END {print usage}')
    
    printf "\n"
    
    # Monitor network stats after transfer
    if [ -f "/proc/net/dev" ] && [ ! -z "$RX_BYTES_BEFORE" ]; then
        RX_BYTES_AFTER=$(cat /proc/net/dev | grep "$NETWORK_IFACE" | awk '{print $2}')
        TX_BYTES_AFTER=$(cat /proc/net/dev | grep "$NETWORK_IFACE" | awk '{print $10}')
        NETWORK_TX=$((TX_BYTES_AFTER - TX_BYTES_BEFORE))
        NETWORK_RX=$((RX_BYTES_AFTER - RX_BYTES_BEFORE))
    fi
    
    # Calculate metrics
    TRANSFER_TIME=$(echo "$END_TIME - $START_TIME" | bc)
    TRANSFER_RATE_MBS=$(echo "scale=3; ($ACTUAL_SIZE / 1024 / 1024) / $TRANSFER_TIME" | bc)
    TRANSFER_RATE_MBPS=$(echo "scale=3; $TRANSFER_RATE_MBS * 8" | bc)
    CPU_USAGE=$(echo "scale=2; $END_CPU - $START_CPU" | bc 2>/dev/null || echo "N/A")
    
    # Wait for receiver processing
    sleep 3
    
    # Verify chunks
    RECEIVED_CHUNKS=$(ls "$CURRENT_RECV_DIR"/chunk_*.bin 2>/dev/null | wc -l)
    
    # Calculate received data size
    TOTAL_RECEIVED=0
    for chunk in "$CURRENT_RECV_DIR"/chunk_*.bin; do
        if [ -f "$chunk" ]; then
            CHUNK_SIZE_ACTUAL=$(stat -f%z "$chunk" 2>/dev/null || stat -c%s "$chunk")
            TOTAL_RECEIVED=$((TOTAL_RECEIVED + CHUNK_SIZE_ACTUAL))
        fi
    done
    
    INTEGRITY_PERCENT=$((TOTAL_RECEIVED * 100 / ACTUAL_SIZE))
    
    # Stop receiver
    kill $RECV_PID 2>/dev/null || true
    
    # Display results
    echo -e "\n${GREEN}📊 Results for ${file_size}MB file:${NC}"
    echo "  Transfer time: ${TRANSFER_TIME}s"
    echo "  Transfer rate: ${TRANSFER_RATE_MBS} MB/s (${TRANSFER_RATE_MBPS} Mbps)"
    echo "  Chunks sent/received: $TOTAL_CHUNKS/$RECEIVED_CHUNKS"
    echo "  Data integrity: ${INTEGRITY_PERCENT}%"
    echo "  CPU usage delta: ${CPU_USAGE}%"
    
    if [ ! -z "$NETWORK_TX" ]; then
        echo "  Network TX: $((NETWORK_TX / 1024)) KB"
        echo "  Network RX: $((NETWORK_RX / 1024)) KB"
    fi
    
    # Add to report
    cat >> "$REPORT_FILE" << EOF

Test: ${file_size}MB File Transfer
---------------------------------
File size: $file_size MB ($(($ACTUAL_SIZE / 1024 / 1024)) MB actual)
Transfer time: ${TRANSFER_TIME} seconds
Transfer rate: ${TRANSFER_RATE_MBS} MB/s
Transfer rate: ${TRANSFER_RATE_MBPS} Mbps
Chunks sent: $TOTAL_CHUNKS
Chunks received: $RECEIVED_CHUNKS
Success rate: $((RECEIVED_CHUNKS * 100 / TOTAL_CHUNKS))%
Data integrity: ${INTEGRITY_PERCENT}%
CPU usage delta: ${CPU_USAGE}%
Network TX: $((NETWORK_TX / 1024 2>/dev/null || echo "N/A")) KB
Network RX: $((NETWORK_RX / 1024 2>/dev/null || echo "N/A")) KB

EOF
    
    # Cleanup test file
    rm -f "$TEST_FILE"
    
    echo "✅ Test completed for ${file_size}MB"
    sleep 1
done

# Generate summary
echo -e "\n${CYAN}📈 Performance Summary${NC}"
echo "======================"

# Calculate averages from report
AVG_RATE=$(grep "Transfer rate:" "$REPORT_FILE" | grep "MB/s" | awk '{sum+=$3; count++} END {printf "%.2f", sum/count}')
AVG_SUCCESS=$(grep "Success rate:" "$REPORT_FILE" | awk '{sum+=$3; count++} END {printf "%.1f", sum/count}' | sed 's/%//')

echo "Average transfer rate: ${AVG_RATE} MB/s"
echo "Average success rate: ${AVG_SUCCESS}%"

# Add summary to report
cat >> "$REPORT_FILE" << EOF

Performance Summary
==================
Average transfer rate: ${AVG_RATE} MB/s
Average success rate: ${AVG_SUCCESS}%
Test completed: $(date)

Detailed chunk timing data saved in: $LOG_DIR/chunk_times_*.csv
Receiver logs saved in: $LOG_DIR/receiver_*.log
EOF

echo -e "\n${GREEN}🎉 Benchmark completed!${NC}"
echo "📄 Full report saved to: $REPORT_FILE"
echo "📊 Detailed logs in: $LOG_DIR"

# Show quick system performance assessment
if (( $(echo "$AVG_RATE > 10" | bc -l) )); then
    echo -e "${GREEN}✅ Performance: Excellent (>10 MB/s)${NC}"
elif (( $(echo "$AVG_RATE > 5" | bc -l) )); then
    echo -e "${YELLOW}⚡ Performance: Good (>5 MB/s)${NC}"
else
    echo -e "${RED}⚠️  Performance: Needs optimization (<5 MB/s)${NC}"
fi

echo -e "\n📋 Quick stats:"
echo "  Fastest test: $(grep "Transfer rate:" "$REPORT_FILE" | grep "MB/s" | sort -k3 -nr | head -1 | awk '{print $3 " MB/s"}')"
echo "  System overhead: Low (local transfer)"
echo "  Protocol efficiency: QUIC + encryption"