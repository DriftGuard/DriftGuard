#!/usr/bin/env python3
"""
Test script for Slack integration with DriftGuard chatbot.
This script tests the Slack tools without requiring the full chatbot system.
"""

import os
import sys
from dotenv import load_dotenv

# Load environment variables
load_dotenv()

# Add the src directory to the path
sys.path.append(os.path.join(os.path.dirname(__file__), 'src'))

from tools.slack_tool import send_drift_report_to_slack, send_drift_alert_to_slack, send_drift_summary_to_slack

def test_slack_configuration():
    """Test if Slack webhook URL is configured."""
    webhook_url = os.getenv("SLACK_WEBHOOK_URL")
    
    print("🔧 Testing Slack Configuration...")
    print(f"Webhook URL configured: {'✅ Yes' if webhook_url else '❌ No'}")
    
    if webhook_url:
        print(f"Webhook URL: {webhook_url[:20]}...{webhook_url[-10:] if len(webhook_url) > 30 else ''}")
    else:
        print("⚠️  Please set SLACK_WEBHOOK_URL in your .env file")
        print("   Example: SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/WEBHOOK/URL")
    
    return bool(webhook_url)

def test_drift_report():
    """Test sending a drift report to Slack."""
    print("\n📊 Testing Drift Report...")
    
    test_message = """
🚨 **DriftGuard Test Report**

**Test Results:**
- Configuration drift detected: 2 resources
- Critical alerts: 1
- Warnings: 3
- Resolved: 5

**Affected Resources:**
- deployment/my-app (namespace: production)
- service/my-service (namespace: staging)

This is a test message from DriftGuard integration testing.
"""
    
    result = send_drift_report_to_slack(test_message)
    print(f"Result: {result}")
    return "✅" in result

def test_drift_alert():
    """Test sending a drift alert to Slack."""
    print("\n🚨 Testing Drift Alert...")
    
    result = send_drift_alert_to_slack(
        alert_type="Configuration Drift",
        resource_name="deployment/test-app",
        namespace="production",
        details="Replica count changed from 3 to 5. This change was not reflected in Git repository."
    )
    
    print(f"Result: {result}")
    return "✅" in result

def test_drift_summary():
    """Test sending a drift summary to Slack."""
    print("\n📈 Testing Drift Summary...")
    
    summary_data = """
**DriftGuard Summary - Last 24 Hours**

📊 **Statistics:**
- Total Resources Monitored: 45
- Active Drift: 3 (6.7%)
- Resolved Drift: 12 (26.7%)
- No Drift: 30 (66.7%)

🔍 **Recent Activity:**
- 2 new drift incidents detected
- 5 drift incidents resolved
- 0 critical alerts

🎯 **Health Score: 85/100**
"""
    
    result = send_drift_summary_to_slack(summary_data)
    print(f"Result: {result}")
    return "✅" in result

def main():
    """Run all Slack integration tests."""
    print("🧪 DriftGuard Slack Integration Test")
    print("=" * 50)
    
    # Test configuration
    config_ok = test_slack_configuration()
    
    if not config_ok:
        print("\n❌ Slack configuration not found. Please set up SLACK_WEBHOOK_URL.")
        print("\nTo get a Slack webhook URL:")
        print("1. Go to your Slack workspace")
        print("2. Create a new app or use an existing one")
        print("3. Enable Incoming Webhooks")
        print("4. Create a webhook for your desired channel")
        print("5. Copy the webhook URL to your .env file")
        return False
    
    # Run tests
    tests = [
        ("Drift Report", test_drift_report),
        ("Drift Alert", test_drift_alert),
        ("Drift Summary", test_drift_summary)
    ]
    
    results = []
    for test_name, test_func in tests:
        try:
            success = test_func()
            results.append((test_name, success))
        except Exception as e:
            print(f"❌ {test_name} failed with error: {e}")
            results.append((test_name, False))
    
    # Summary
    print("\n" + "=" * 50)
    print("📋 Test Results Summary:")
    print("=" * 50)
    
    passed = 0
    for test_name, success in results:
        status = "✅ PASS" if success else "❌ FAIL"
        print(f"{test_name}: {status}")
        if success:
            passed += 1
    
    print(f"\nOverall: {passed}/{len(results)} tests passed")
    
    if passed == len(results):
        print("🎉 All tests passed! Slack integration is working correctly.")
    else:
        print("⚠️  Some tests failed. Check your Slack webhook configuration.")
    
    return passed == len(results)

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)
