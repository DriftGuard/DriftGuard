import requests
from typing import Dict, Any, Optional
from langchain_core.tools import tool
import json

class DriftGuardAPI:
    """DriftGuard API client for fetching drift statistics and records."""
    
    def __init__(self, base_url: str = "http://localhost:8080"):
        self.base_url = base_url
        
    def _make_request(self, endpoint: str) -> Optional[Dict[str, Any]]:
        """Make a request to DriftGuard API."""
        try:
            response = requests.get(f"{self.base_url}{endpoint}", timeout=10)
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException as e:
            print(f"Error making request to {endpoint}: {e}")
            return None
    
    def get_health(self) -> Optional[Dict[str, Any]]:
        """Get DriftGuard health status."""
        return self._make_request("/health")
    
    def get_statistics(self) -> Optional[Dict[str, Any]]:
        """Get drift statistics."""
        return self._make_request("/api/v1/statistics")
    
    def get_drift_records(self) -> Optional[Dict[str, Any]]:
        """Get all drift records."""
        return self._make_request("/api/v1/drift-records")
    
    def get_active_drift(self) -> Optional[Dict[str, Any]]:
        """Get active drift records."""
        return self._make_request("/api/v1/drift-records/active")
    
    def get_resolved_drift(self) -> Optional[Dict[str, Any]]:
        """Get resolved drift records."""
        return self._make_request("/api/v1/drift-records/resolved")
    
    def trigger_analysis(self) -> Optional[Dict[str, Any]]:
        """Trigger manual drift analysis."""
        try:
            response = requests.post(f"{self.base_url}/api/v1/analyze", timeout=10)
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException as e:
            print(f"Error triggering analysis: {e}")
            return None

# Initialize the DriftGuard API client
driftguard_api = DriftGuardAPI()

@tool
def get_drift_statistics() -> str:
    """
    Get comprehensive drift statistics from DriftGuard.
    Returns information about total records, active drift, resolved drift, and percentages.
    """
    stats = driftguard_api.get_statistics()
    if not stats:
        return "❌ Failed to fetch drift statistics. DriftGuard service may be unavailable."
    
    statistics = stats.get("statistics", {})
    
    # Format the statistics in a readable way
    total = statistics.get("total_records", 0)
    active = statistics.get("active_drift", 0)
    resolved = statistics.get("resolved_drift", 0)
    no_drift = statistics.get("no_drift", 0)
    active_pct = statistics.get("active_percentage", 0)
    resolved_pct = statistics.get("resolved_percentage", 0)
    
    result = f"""
📊 **DriftGuard Statistics Summary**

🔍 **Overview:**
- Total Records: {total}
- Active Drift: {active} ({active_pct:.1f}%)
- Resolved Drift: {resolved} ({resolved_pct:.1f}%)
- No Drift: {no_drift}

⚠️ **Current Status:**
- Recent Active Drift: {statistics.get("recent_active_drift", 0)}
- Recent Resolutions: {statistics.get("recent_resolutions", 0)}

🚨 **Alert Level:** {"HIGH" if active_pct > 50 else "MEDIUM" if active_pct > 20 else "LOW"}
"""
    return result

@tool
def get_active_drift_details() -> str:
    """
    Get detailed information about currently active configuration drift.
    Shows which resources have configuration drift and what changes were detected.
    """
    active_drift = driftguard_api.get_active_drift()
    if not active_drift:
        return "❌ Failed to fetch active drift records. DriftGuard service may be unavailable."
    
    records = active_drift.get("drift_records", [])
    count = active_drift.get("count", 0)
    
    if count == 0:
        return "✅ **No Active Configuration Drift Detected**\n\nAll monitored resources are in sync with their desired state in Git."
    
    result = f"🚨 **Active Configuration Drift Detected** ({count} resources)\n\n"
    
    for i, record in enumerate(records[:5], 1):  # Limit to first 5 records
        resource_id = record.get("resource_id", "Unknown")
        kind = record.get("kind", "Unknown")
        namespace = record.get("namespace", "Unknown")
        name = record.get("name", "Unknown")
        first_detected = record.get("first_detected", "Unknown")
        
        drift_details = record.get("drift_details", [])
        
        result += f"**{i}. {kind}: {name}** (Namespace: {namespace})\n"
        result += f"   - Resource ID: {resource_id}\n"
        result += f"   - First Detected: {first_detected}\n"
        
        if drift_details:
            result += "   - Changes Detected:\n"
            for detail in drift_details[:3]:  # Limit to first 3 changes
                field = detail.get("field", "Unknown")
                from_val = detail.get("from", "Unknown")
                to_val = detail.get("to", "Unknown")
                severity = detail.get("severity", "unknown")
                result += f"     • {field}: {from_val} → {to_val} (Severity: {severity})\n"
        
        result += "\n"
    
    if count > 5:
        result += f"... and {count - 5} more records. Use the DriftGuard API directly for complete details.\n"
    
    return result

@tool
def get_drift_health_check() -> str:
    """
    Check if DriftGuard service is healthy and responsive.
    """
    health = driftguard_api.get_health()
    if not health:
        return "❌ **DriftGuard Service is DOWN**\n\nThe DriftGuard service is not responding. Please check if the service is running on http://localhost:8080"
    
    status = health.get("status", "unknown")
    message = health.get("message", "No message")
    timestamp = health.get("time", "Unknown")
    
    if status == "healthy":
        return f"✅ **DriftGuard Service is HEALTHY**\n\nStatus: {status}\nMessage: {message}\nLast Check: {timestamp}"
    else:
        return f"⚠️ **DriftGuard Service Status: {status}**\n\nMessage: {message}\nLast Check: {timestamp}"

@tool
def trigger_drift_analysis() -> str:
    """
    Trigger a manual drift analysis to detect configuration changes.
    This will compare current Kubernetes state with Git repository state.
    """
    result = driftguard_api.trigger_analysis()
    if not result:
        return "❌ Failed to trigger drift analysis. DriftGuard service may be unavailable."
    
    status = result.get("status", "unknown")
    message = result.get("message", "No message")
    
    return f"🔄 **Drift Analysis Triggered**\n\nStatus: {status}\nMessage: {message}\n\nThe analysis is now running in the background. Check drift statistics in a few moments for updated results."

@tool
def get_comprehensive_drift_report() -> str:
    """
    Get a comprehensive drift report including health, statistics, and active drift details.
    This provides a complete overview of the DriftGuard system status.
    """
    # Get health status
    health_result = get_drift_health_check()
    
    # Get statistics
    stats_result = get_drift_statistics()
    
    # Get active drift
    active_result = get_active_drift_details()
    
    return f"""
🎯 **COMPREHENSIVE DRIFTGUARD REPORT**

{health_result}

{stats_result}

{active_result}

💡 **Recommendations:**
- Monitor active drift regularly
- Investigate high-severity drifts immediately
- Consider triggering manual analysis if needed
- Review resolved drifts to prevent recurrence
"""

# Import Slack tools
from .slack_tool import slack_tools

# Export all tools for easy import
drift_tools = [
    get_drift_statistics,
    get_active_drift_details,
    get_drift_health_check,
    trigger_drift_analysis,
    get_comprehensive_drift_report
] + slack_tools
