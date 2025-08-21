from typing_extensions import TypedDict
from langgraph.graph.message import add_messages
from typing import Annotated

class State(TypedDict):
    '''
    represents the structure of the state of the graph

    '''

    messages: Annotated[list, add_messages]
    