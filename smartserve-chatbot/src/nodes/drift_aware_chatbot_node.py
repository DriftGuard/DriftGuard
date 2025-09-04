from langchain_core.messages import AIMessage, SystemMessage
from src.states.state import State
from src.tools.driftguard_tool import drift_tools
from langchain_core.tools import tool
from typing import List, Any

class DriftAwareChatbotNode:
    """
    Enhanced chatbot node that can interact with DriftGuard APIs and provide 
    intelligent responses about configuration drift monitoring.
    """
    
    def __init__(self, llm):
        self.llm = llm
        self.system_prompt = """You are DriftGuard Assistant, an intelligent AI helper specialized in GitOps configuration drift monitoring and Kubernetes infrastructure management.

**Your Role:**
- Help users understand and manage configuration drift in their Kubernetes clusters
- Provide insights about DriftGuard monitoring system
- Explain drift detection results and recommend actions
- Guide users through GitOps best practices

**Available Tools:**
You have access to several DriftGuard tools:
1. get_drift_statistics() - Get comprehensive drift statistics
2. get_active_drift_details() - Get detailed information about active drift
3. get_drift_health_check() - Check DriftGuard service health
4. trigger_drift_analysis() - Trigger manual drift analysis
5. get_comprehensive_drift_report() - Get complete system overview
6. send_drift_report_to_slack() - Send DriftGuard reports to Slack
7. send_drift_alert_to_slack() - Send specific drift alerts to Slack
8. send_drift_summary_to_slack() - Send comprehensive drift summaries to Slack

**Response Style:**
- Be concise and actionable
- Use emojis and formatting for clarity
- Explain technical concepts in accessible terms
- Always provide context about what drift means and why it matters
- Suggest specific next steps when appropriate

**Key Topics You Can Help With:**
- Configuration drift explanation and analysis
- GitOps vs InfraOps comparison
- Kubernetes resource monitoring
- Drift resolution strategies
- Best practices for infrastructure as code
- DriftGuard system status and health
- Slack notifications and alerting setup
- Team communication and incident response

**Slack Integration:**
- Automatically send drift reports to Slack channels
- Send targeted alerts for specific drift incidents
- Share comprehensive drift summaries with teams
- Requires SLACK_WEBHOOK_URL environment variable configuration

When users ask about drift, infrastructure, monitoring, or related topics, proactively use the available tools to provide current, real-time information."""

    def process(self, state: State) -> State:
        """
        Process the user message and provide intelligent responses about DriftGuard and configuration drift.
        """
        # Get the latest message
        messages = state["messages"]
        latest_message = messages[-1] if messages else None
        
        if not latest_message:
            return state
        
        # Create a tool-enabled LLM
        llm_with_tools = self.llm.bind_tools(drift_tools)
        
        # Prepare the conversation with system prompt
        conversation_messages = [
            SystemMessage(content=self.system_prompt),
            *messages
        ]
        
        # Get response from LLM
        response = llm_with_tools.invoke(conversation_messages)
        
        # If the LLM wants to use tools, execute them
        if response.tool_calls:
            # Execute each tool call
            tool_results = []
            for tool_call in response.tool_calls:
                tool_name = tool_call["name"]
                
                # Find and execute the tool
                for tool in drift_tools:
                    if tool.name == tool_name:
                        try:
                            result = tool.invoke(tool_call.get("args", {}))
                            tool_results.append(f"**{tool_name} Result:**\n{result}")
                        except Exception as e:
                            tool_results.append(f"**{tool_name} Error:** {str(e)}")
                        break
            
            # Create a follow-up response with tool results
            tool_results_text = "\n\n".join(tool_results)
            
            # Get final response incorporating tool results
            final_conversation = [
                SystemMessage(content=self.system_prompt),
                *messages,
                AIMessage(content=f"I'll help you with that. Let me check the current DriftGuard status:\n\n{tool_results_text}"),
            ]
            
            final_response = self.llm.invoke(final_conversation)
            state["messages"].append(final_response)
        else:
            # No tools needed, just add the response
            state["messages"].append(response)
        
        return state

class BasicDriftChatbotNode:
    """
    Simplified version for basic drift monitoring questions without complex tool orchestration.
    """
    
    def __init__(self, llm):
        self.llm = llm
        self.drift_context = """You are a DriftGuard Assistant specializing in GitOps configuration drift monitoring.

DriftGuard is an intelligent system that:
- Monitors Kubernetes resources for configuration drift
- Compares live cluster state with desired state in Git repositories  
- Detects when manual changes diverge from GitOps principles
- Tracks drift history and resolution status
- Provides alerts and insights for infrastructure teams

Common drift scenarios:
- Manual kubectl scaling (changing replicas)
- Direct resource edits bypassing Git
- Environment-specific modifications
- Resource updates outside GitOps workflow

You can help explain concepts, analyze drift data, and recommend GitOps best practices."""

    def process(self, state: State) -> State:
        """
        Process basic drift-related queries with context about DriftGuard.
        """
        messages = state["messages"]
        
        # Add drift context to the conversation
        context_message = SystemMessage(content=self.drift_context)
        
        # Prepare conversation with context
        conversation = [context_message] + messages
        
        # Get response
        response = self.llm.invoke(conversation)
        state["messages"].append(response)
        
        return state


