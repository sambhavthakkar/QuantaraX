#!/bin/bash

# Simple QuantaraX Performance Test
echo "🚀 QuantaraX Simple Performance Test"
echo "===================================="

# Test with different file sizes
for size in 1 5 10; do
    echo -e "\n📊 Testing ${size}MB file transfer"
    echo "--------------------------------"
    
    # Create test file
    dd if=/dev/urandom of="test_${size}mb.bin" bs=1M count=$size 2>/dev/null
    
    # Start receiver
    PORT=$((44500 + size))
    RECV_DIR="recv_${size}mb"
    mkdir -p "$RECV_DIR"
    
    ./bin/quic_recv --listen "localhost:$PORT" --output-dir "$RECV_DIR" > "recv_${size}mb.log" 2>&1 &
    RECV_PID=$!
    sleep 1
    
    # Transfer with timing
    echo "⏱️  Starting transfer..."
    START_TIME=$(date +%s.%N)
    
    ./bin/quic_send \
        --addr "localhost:$PORT" \
        --file "test_${size}mb.bin" \
        --chunk-index 0 \
        --chunk-size 1048576 > "send_${size}mb.log" 2>&1
    
    END_TIME=$(date +%s.%N)
    
    # Calculate performance
    DURATION=$(echo "$END_TIME - $START_TIME" | bc)
    SPEED_MBS=$(echo "scale=2; $size / $DURATION" | bc)
    SPEED_MBPS=$(echo "scale=2; $SPEED_MBS * 8" | bc)
    
    sleep 1
    kill $RECV_PID 2>/dev/null || true
    
    # Check if file was received
    if [ -f "$RECV_DIR/chunk_0000.bin" ]; then
        RECV_SIZE=$(stat -f%z "$RECV_DIR/chunk_0000.bin" 2>/dev/null || stat -c%s "$RECV_DIR/chunk_0000.bin")
        RECV_MB=$(echo "scale=2; $RECV_SIZE / 1024 / 1024" | bc)
        echo "✅ Transfer successful!"
        echo "   Duration: ${DURATION}s"
        echo "   Speed: ${SPEED_MBS} MB/s (${SPEED_MBPS} Mbps)"
        echo "   File size: ${RECV_MB} MB received"
    else
        echo "❌ Transfer failed"
    fi
    
    # Cleanup
    rm -f "test_${size}mb.bin"
done

echo -e "\n🎉 Performance test completed!"