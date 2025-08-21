from src.states.state import State


class BasicChatbotNode:
    """
    Basic chatbot logic node
    """

    def __init__(self, model):
        self.llm = model

    def process(self, state: State)->dict:
        """
        process the user input and return the response
        """
        return {"messages": self.llm.invoke(state['messages'])}