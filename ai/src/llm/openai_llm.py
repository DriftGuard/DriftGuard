import os
from langchain_openai import ChatOpenAI


class OpenAILLM:
    def __init__(self):
        self.user_controls_input = {
            "OPENAI_API_KEY": os.getenv("OPENAI_API_KEY", ""),
            "selected_model": os.getenv("OPENAI_MODEL", "gpt-4o-mini"),
        }

    def get_llm_model(self):
        try:
            openai_api_key = self.user_controls_input["OPENAI_API_KEY"]
            selected_openai_model = self.user_controls_input["selected_model"]
            if not openai_api_key:
                raise ValueError("OpenAI API key is required to use OpenAI models.")

            llm = ChatOpenAI(api_key=openai_api_key, model=selected_openai_model)

        except Exception as e:
            raise ValueError(f"Error initializing OpenAI LLM: {e}")
        return llm


