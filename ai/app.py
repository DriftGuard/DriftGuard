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

if __name__=="__main__":
    uvicorn.run("app:app",host="0.0.0.0",port=8000,reload=True)