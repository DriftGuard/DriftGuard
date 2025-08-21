#!/usr/bin/env python3
"""
Test script for DriftGuard + Chatbot Integration

This script demonstrates how to interact with the enhanced chatbot that can
access DriftGuard APIs and provide intelligent responses about configuration drift.

Prerequisites:
1. DriftGuard service running on http://localhost:8080
2. OpenAI API key set in .env file
3. Chatbot service running on http://localhost:8000
"""

import requests
import json
import time
from typing import Dict, Any

class DriftGuardChatbotTester:
    def __init__(self, chatbot_url: str = "http://localhost:8000", driftguard_url: str = "http://localhost:8080"):
        self.chatbot_url = chatbot_url
        self.driftguard_url = driftguard_url
    
    def test_drift_status_endpoint(self) -> Dict[str, Any]:
        """Test the direct drift status endpoint."""
        print("🔍 Testing Direct Drift Status Endpoint...")
        try:
            response = requests.get(f"{self.chatbot_url}/drift-status", timeout=10)
            response.raise_for_status()
            data = response.json()
            print("✅ Drift Status Retrieved Successfully!")
            print(json.dumps(data, indent=2))
            return data
        except Exception as e:
            print(f"❌ Error: {e}")
            return {}
    
    def test_basic_drift_chatbot(self, query: str) -> Dict[str, Any]:
        """Test the basic drift chatbot (no real-time API calls)."""
        print(f"\n💬 Testing Basic Drift Chatbot with query: '{query}'")
        try:
            payload = {"topic": query}
            response = requests.post(f"{self.chatbot_url}/basic-drift-query", json=payload, timeout=30)
            response.raise_for_status()
            data = response.json()
            
            if "data" in data:
                messages = data["data"].get("messages", [])
                if messages:
                    last_message = messages[-1]
                    print("🤖 Chatbot Response:")
                    print(last_message.get("content", "No content"))
            
            return data
        except Exception as e:
            print(f"❌ Error: {e}")
            return {}
    
    def test_drift_aware_chatbot(self, query: str) -> Dict[str, Any]:
        """Test the advanced drift-aware chatbot (with real-time API calls)."""
        print(f"\n🧠 Testing Advanced Drift-Aware Chatbot with query: '{query}'")
        try:
            payload = {"topic": query}
            response = requests.post(f"{self.chatbot_url}/drift-query", json=payload, timeout=60)
            response.raise_for_status()
            data = response.json()
            
            if "data" in data:
                messages = data["data"].get("messages", [])
                if messages:
                    last_message = messages[-1]
                    print("🤖 Advanced Chatbot Response:")
                    print(last_message.get("content", "No content"))
            
            return data
        except Exception as e:
            print(f"❌ Error: {e}")
            return {}
    
    def test_driftguard_health(self) -> bool:
        """Check if DriftGuard service is healthy."""
        print("\n🏥 Checking DriftGuard Health...")
        try:
            response = requests.get(f"{self.driftguard_url}/health", timeout=10)
            response.raise_for_status()
            health_data = response.json()
            print(f"✅ DriftGuard is healthy: {health_data}")
            return True
        except Exception as e:
            print(f"❌ DriftGuard health check failed: {e}")
            return False
    
    def run_comprehensive_test(self):
        """Run a comprehensive test of the DriftGuard + Chatbot integration."""
        print("🚀 Starting Comprehensive DriftGuard + Chatbot Integration Test\n")
        print("=" * 60)
        
        # Test 1: Check DriftGuard health
        if not self.test_driftguard_health():
            print("⚠️ Warning: DriftGuard service is not healthy. Some tests may fail.")
        
        # Test 2: Test direct drift status endpoint
        self.test_drift_status_endpoint()
        
        # Test 3: Test basic drift chatbot with conceptual questions
        basic_queries = [
            "What is configuration drift?",
            "Explain GitOps vs InfraOps",
            "How does DriftGuard work?",
            "What are the benefits of drift detection?"
        ]
        
        for query in basic_queries:
            self.test_basic_drift_chatbot(query)
            time.sleep(1)  # Small delay between requests
        
        # Test 4: Test advanced drift-aware chatbot with real-time queries
        advanced_queries = [
            "What's the current drift status?",
            "Show me the drift statistics",
            "Are there any active configuration drifts?",
            "Give me a comprehensive drift report",
            "Check DriftGuard health status"
        ]
        
        for query in advanced_queries:
            self.test_drift_aware_chatbot(query)
            time.sleep(2)  # Longer delay for API calls
        
        print("\n" + "=" * 60)
        print("🎉 Comprehensive Test Complete!")
        print("\nNext Steps:")
        print("1. Try asking the chatbot about specific drift scenarios")
        print("2. Test with different types of configuration changes")
        print("3. Explore the chatbot's ability to explain technical concepts")

def main():
    """Main test execution."""
    print("DriftGuard + LangChain Chatbot Integration Tester")
    print("=" * 50)
    
    tester = DriftGuardChatbotTester()
    
    # Quick health checks
    print("Performing initial health checks...")
    
    # Check if chatbot is running
    try:
        response = requests.get("http://localhost:8000", timeout=5)
        print("✅ Chatbot service is running")
    except:
        print("❌ Chatbot service is not running. Start it with: python app.py")
        return
    
    # Check if DriftGuard is running
    if not tester.test_driftguard_health():
        print("⚠️ DriftGuard service is not running. Some features will be limited.")
    
    print("\nStarting integration tests...\n")
    tester.run_comprehensive_test()

if __name__ == "__main__":
    main()
