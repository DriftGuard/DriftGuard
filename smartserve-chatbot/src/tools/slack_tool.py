from langchain_core.tools import tool
import os
import requests
from datetime import datetime
from typing import Optional

class SlackTool:
    """Slack tool for sending security analysis reports to Slack"""
    
    def __init__(self):
        self.webhook_url = os.getenv("SLACK_WEBHOOK_URL")

    def format_security_report_for_slack(self, report_content: str) -> str:
        """Format the security report for better Slack presentation"""
        # Truncate if too long for Slack (max 3000 chars for text blocks)
        if len(report_content) > 2800:
            report_content = report_content[:2800] + "\n...\n[Report truncated - see full details in logs]"
        
        return report_content

    def send_to_slack(self, message: str) -> bool:
        """Send security analysis report to Slack"""
        if not self.webhook_url:
            self.webhook_url = os.getenv("SLACK_WEBHOOK_URL")
        
        if not self.webhook_url:
            print("⚠️ No Slack webhook URL found. Set SLACK_WEBHOOK_URL in .env file")
            return False
        
        try:
            # Format message for Slack
            slack_payload = {
                "text": "🔒 Security Analysis Report",
                "blocks": [
                    {
                        "type": "header",
                        "text": {
                            "type": "plain_text",
                            "text": "🔒 Security Analysis Report"
                        }
                    },
                    {
                        "type": "section",
                        "fields": [
                            {
                                "type": "mrkdwn",
                                "text": f"*Timestamp:* {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}"
                            },
                            {
                                "type": "mrkdwn",
                                "text": f"*Analysis Type:* Automated Security Scan"
                            }
                        ]
                    },
                    {
                        "type": "section",
                        "text": {
                            "type": "mrkdwn",
                            "text": f"```\n{self.format_security_report_for_slack(message)}\n```"
                        }
                    }
                ]
            }
            
            response = requests.post(
                self.webhook_url,
                json=slack_payload,
                headers={'Content-Type': 'application/json'},
                timeout=10
            )
            
            if response.status_code == 200:
                print("✅ Security report sent to Slack successfully!")
                return True
            else:
                print(f"❌ Failed to send to Slack. Status: {response.status_code}")
                return False
                
        except Exception as e:
            print(f"❌ Error sending to Slack: {str(e)}")
            return False

# Initialize the Slack tool instance
slack_tool_instance = SlackTool()

@tool
def send_drift_report_to_slack(message: str) -> str:
    """
    Send a DriftGuard security analysis report to Slack.
    Requires SLACK_WEBHOOK_URL environment variable to be set.
    """
    success = slack_tool_instance.send_to_slack(message)
    
    if success:
        return "✅ DriftGuard report successfully sent to Slack!"
    else:
        return "❌ Failed to send DriftGuard report to Slack. Check webhook URL configuration."

@tool
def send_drift_alert_to_slack(alert_type: str, resource_name: str, namespace: str, details: str) -> str:
    """
    Send a specific drift alert to Slack with structured information.
    
    Args:
        alert_type: Type of drift detected (e.g., "Configuration Drift", "Resource Change")
        resource_name: Name of the affected resource
        namespace: Kubernetes namespace
        details: Detailed information about the drift
    """
    if not slack_tool_instance.webhook_url:
        return "❌ Slack webhook URL not configured. Set SLACK_WEBHOOK_URL environment variable."
    
    # Create a structured alert message
    alert_message = f"""
🚨 **DriftGuard Alert: {alert_type}**

**Resource:** {resource_name}
**Namespace:** {namespace}
**Timestamp:** {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}

**Details:**
{details}

**Action Required:** Please investigate and resolve this configuration drift.
"""
    
    success = slack_tool_instance.send_to_slack(alert_message)
    
    if success:
        return f"✅ Drift alert for {resource_name} sent to Slack successfully!"
    else:
        return f"❌ Failed to send drift alert for {resource_name} to Slack."

@tool
def send_drift_summary_to_slack(summary_data: str) -> str:
    """
    Send a comprehensive drift summary report to Slack.
    
    Args:
        summary_data: Formatted summary of drift statistics and status
    """
    if not slack_tool_instance.webhook_url:
        return "❌ Slack webhook URL not configured. Set SLACK_WEBHOOK_URL environment variable."
    
    # Create a comprehensive summary message
    summary_message = f"""
📊 **DriftGuard Summary Report**

{summary_data}

**Report Generated:** {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}
"""
    
    success = slack_tool_instance.send_to_slack(summary_message)
    
    if success:
        return "✅ DriftGuard summary report sent to Slack successfully!"
    else:
        return "❌ Failed to send DriftGuard summary report to Slack."

# Export all Slack tools for easy import
slack_tools = [
    send_drift_report_to_slack,
    send_drift_alert_to_slack,
    send_drift_summary_to_slack
]