#!/bin/bash

# Test script for the restaurant order system

echo "=== Restaurant Order System Test ==="
echo

# Wait for services to be ready
echo "Waiting for services to start..."
sleep 10

# Test 1: Create a takeout order
echo "1. Creating a takeout order..."
curl -X POST http://localhost:3000/orders \
  -H "Content-Type: application/json" \
  -d '{
        "customer_name": "John Doe",
        "order_type": "takeout",
        "items": [
          {"name": "Margherita Pizza", "quantity": 1, "price": 15.99},
          {"name": "Caesar Salad", "quantity": 1, "price": 8.99}
        ]
      }'
echo -e "\n"

# Test 2: Create a delivery order
echo "2. Creating a delivery order..."
curl -X POST http://localhost:3000/orders \
  -H "Content-Type: application/json" \
  -d '{
        "customer_name": "Jane Smith",
        "order_type": "delivery",
        "delivery_address": "123 Main Street, Downtown",
        "items": [
          {"name": "Pepperoni Pizza", "quantity": 2, "price": 18.99},
          {"name": "Garlic Bread", "quantity": 1, "price": 5.99}
        ]
      }'
echo -e "\n"

# Test 3: Create a dine-in order
echo "3. Creating a dine-in order..."
curl -X POST http://localhost:3000/orders \
  -H "Content-Type: application/json" \
  -d '{
        "customer_name": "Bob Wilson",
        "order_type": "dine_in",
        "table_number": 5,
        "items": [
          {"name": "Hawaiian Pizza", "quantity": 1, "price": 17.99},
          {"name": "Coke", "quantity": 2, "price": 2.99}
        ]
      }'
echo -e "\n"

# Test 4: Check health endpoint
echo "4. Checking order service health..."
curl http://localhost:3000/health
echo -e "\n"

echo
echo "=== Test completed ==="
echo "Check the logs of kitchen workers and notification subscriber to see message processing."
echo "You can also view RabbitMQ Management UI at http://localhost:15672 (guest/guest)"
