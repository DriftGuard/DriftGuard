import uvicorn
from fastapi import FastAPI, Request
from src.graphs.graph_builder import GraphBuilder
from src.llm.openai_llm import OpenAILLM
from langchain_core.messages import AIMessage, HumanMessage, SystemMessage
import os
from dotenv import load_dotenv
load_dotenv()

app=FastAPI()

os.environ["LANGSMITH_TRACING_V2"]="true"
os.environ["LANGSMITH_API_KEY"]=os.getenv("LANGSMITH_API_KEY")

## API's

@app.post("/query")
async def Chatbot(request:Request):
    data=await request.json()
    topic= data.get("topic","")

    ## get LLM object
    openai_llm = OpenAILLM()
    llm = openai_llm.get_llm_model()

    #get graph
    graph_builder=GraphBuilder(llm)
    if topic:
        graph=graph_builder.basic_chatbot_build_graph()
        state=graph.invoke({"messages": [HumanMessage(content=topic)]},
    config={"configurable": {"thread_id": "1"}})
    return {"data":state}

@app.post("/drift-query")
async def DriftAwareChatbot(request: Request):
    """
    Enhanced chatbot endpoint that can access DriftGuard APIs and provide
    intelligent responses about configuration drift monitoring.
    """
    data = await request.json()
    topic = data.get("topic", "")
    
    if not topic:
        return {"error": "Topic is required"}
    
    ## get LLM object
    openai_llm = OpenAILLM()
    llm = openai_llm.get_llm_model()
    
    #get drift-aware graph
    graph_builder = GraphBuilder(llm)
    graph = graph_builder.drift_aware_chatbot_build_graph()
    
    try:
        state = graph.invoke(
            {"messages": [HumanMessage(content=topic)]},
            config={"configurable": {"thread_id": "drift_session"}}
        )
        return {"data": state}
    except Exception as e:
        return {"error": f"Error processing drift query: {str(e)}"}

@app.post("/basic-drift-query")
async def BasicDriftChatbot(request: Request):
    """
    Basic drift-aware chatbot that provides context about DriftGuard
    without real-time API integration.
    """
    data = await request.json()
    topic = data.get("topic", "")
    
    if not topic:
        return {"error": "Topic is required"}
    
    ## get LLM object
    openai_llm = OpenAILLM()
    llm = openai_llm.get_llm_model()
    
    #get basic drift graph
    graph_builder = GraphBuilder(llm)
    graph = graph_builder.basic_drift_chatbot_build_graph()
    
    try:
        state = graph.invoke(
            {"messages": [HumanMessage(content=topic)]},
            config={"configurable": {"thread_id": "basic_drift_session"}}
        )
        return {"data": state}
    except Exception as e:
        return {"error": f"Error processing basic drift query: {str(e)}"}

@app.get("/drift-status")
async def DriftStatus():
    """
    Quick endpoint to get current drift status without chatbot interaction.
    """
    from src.tools.driftguard_tool import driftguard_api
    
    try:
        health = driftguard_api.get_health()
        stats = driftguard_api.get_statistics()
        
        if not health or not stats:
            return {"status": "error", "message": "DriftGuard service unavailable"}
        
        return {
            "status": "success",
            "health": health,
            "statistics": stats.get("statistics", {}),
            "timestamp": health.get("time")
        }
    except Exception as e:
        return {"status": "error", "message": str(e)}

@app.post("/test-slack")
async def TestSlack(request: Request):
    """
    Test endpoint for Slack integration.
    Sends a test message to Slack to verify webhook configuration.
    """
    from src.tools.slack_tool import send_drift_report_to_slack
    
    data = await request.json()
    test_message = data.get("message", "🧪 Test message from DriftGuard chatbot")
    
    try:
        result = send_drift_report_to_slack(test_message)
        return {
            "status": "success" if "✅" in result else "error",
            "message": result,
            "test_message": test_message
        }
    except Exception as e:
        return {"status": "error", "message": f"Slack test failed: {str(e)}"}

@app.post("/send-drift-alert")
async def SendDriftAlert(request: Request):
    """
    Send a drift alert to Slack with specific details.
    """
    from src.tools.slack_tool import send_drift_alert_to_slack
    
    data = await request.json()
    alert_type = data.get("alert_type", "Configuration Drift")
    resource_name = data.get("resource_name", "Unknown Resource")
    namespace = data.get("namespace", "default")
    details = data.get("details", "No details provided")
    
    try:
        result = send_drift_alert_to_slack(alert_type, resource_name, namespace, details)
        return {
            "status": "success" if "✅" in result else "error",
            "message": result,
            "alert_data": {
                "alert_type": alert_type,
                "resource_name": resource_name,
                "namespace": namespace,
                "details": details
            }
        }
    except Exception as e:
        return {"status": "error", "message": f"Failed to send drift alert: {str(e)}"}

if __name__=="__main__":
    uvicorn.run("app:app",host="0.0.0.0",port=8000,reload=True)