from langgraph.graph import StateGraph, START, END
from src.states.state import State
from src.nodes.basic_chatbot_node import BasicChatbotNode
from src.nodes.drift_aware_chatbot_node import DriftAwareChatbotNode, BasicDriftChatbotNode
from langgraph.checkpoint.memory import MemorySaver

# Reuse a single in-memory saver across the process so checkpoints persist between requests
memory_saver = MemorySaver()

class GraphBuilder:
    def __init__(self,model):
        self.llm = model
        self.graph_builder=StateGraph(State)

    def basic_chatbot_build_graph(self):
        """
        Builds a basic chatbot graph with a single node that takes user input and returns a response.
        """
        chatbot_node = BasicChatbotNode(self.llm)
        self.graph_builder.add_node("chatbot", chatbot_node.process)
        self.graph_builder.add_edge(START, "chatbot") 
        self.graph_builder.add_edge("chatbot", END)
        app = self.graph_builder.compile(checkpointer=memory_saver)
        # Compile the graph before returning
        return app
    
    def drift_aware_chatbot_build_graph(self):
        """
        Builds a DriftGuard-aware chatbot graph that can access drift statistics and provide 
        intelligent insights about configuration drift monitoring.
        """
        drift_chatbot_node = DriftAwareChatbotNode(self.llm)
        self.graph_builder.add_node("drift_chatbot", drift_chatbot_node.process)
        self.graph_builder.add_edge(START, "drift_chatbot") 
        self.graph_builder.add_edge("drift_chatbot", END)
        app = self.graph_builder.compile(checkpointer=memory_saver)
        return app
    
    def basic_drift_chatbot_build_graph(self):
        """
        Builds a basic drift-aware chatbot that provides context about DriftGuard 
        without real-time API integration.
        """
        basic_drift_node = BasicDriftChatbotNode(self.llm)
        self.graph_builder.add_node("basic_drift_chatbot", basic_drift_node.process)
        self.graph_builder.add_edge(START, "basic_drift_chatbot") 
        self.graph_builder.add_edge("basic_drift_chatbot", END)
        app = self.graph_builder.compile(checkpointer=memory_saver)
        return app