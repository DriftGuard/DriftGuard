# DriftGuard SmartServe Chatbot

An intelligent AI chatbot service for DriftGuard that provides real-time configuration drift monitoring, analysis, and team communication through Slack integration.

## 🚀 Features

- **DriftGuard Integration**: Real-time access to drift detection APIs
- **Intelligent Analysis**: AI-powered drift analysis and recommendations
- **Slack Notifications**: Send alerts and reports directly to Slack channels
- **Multiple Chatbot Modes**: Basic, drift-aware, and comprehensive reporting
- **REST API**: Easy integration with external systems

## 🛠️ Setup

### Prerequisites

- Python 3.11+
- DriftGuard backend running on localhost:8080
- OpenAI API key
- (Optional) Slack webhook URL for notifications

### Installation

1. **Clone and navigate to the chatbot directory:**
   ```bash
   cd smartserve-chatbot
   ```

2. **Install dependencies:**
   ```bash
   pip install -r requirements.txt
   ```

3. **Configure environment variables:**
   ```bash
   cp config.example.env .env
   # Edit .env with your configuration
   ```

4. **Set up environment variables in `.env`:**
   ```env
   # Required
   OPENAI_API_KEY=your_openai_api_key_here
   
   # Optional - for Slack integration
   SLACK_WEBHOOK_URL=your_slack_webhook_url_here
   
   # Optional - for tracing
   LANGSMITH_API_KEY=your_langsmith_api_key_here
   ```

### Running the Service

```bash
python app.py
```

The service will start on `http://localhost:8000`

## 📡 API Endpoints

### Chatbot Endpoints

- `POST /query` - Basic chatbot interaction
- `POST /drift-query` - Drift-aware chatbot with DriftGuard integration
- `POST /basic-drift-query` - Basic drift context without API calls

### Monitoring Endpoints

- `GET /drift-status` - Quick drift status check
- `POST /test-slack` - Test Slack integration
- `POST /send-drift-alert` - Send specific drift alert to Slack

### Example Usage

**Basic Drift Query:**
```bash
curl -X POST "http://localhost:8000/drift-query" \
  -H "Content-Type: application/json" \
  -d '{"topic": "What is the current drift status?"}'
```

**Test Slack Integration:**
```bash
curl -X POST "http://localhost:8000/test-slack" \
  -H "Content-Type: application/json" \
  -d '{"message": "Test message from DriftGuard"}'
```

**Send Drift Alert:**
```bash
curl -X POST "http://localhost:8000/send-drift-alert" \
  -H "Content-Type: application/json" \
  -d '{
    "alert_type": "Configuration Drift",
    "resource_name": "deployment/my-app",
    "namespace": "production",
    "details": "Replica count changed from 3 to 5"
  }'
```

## 🔧 Slack Integration

The chatbot includes comprehensive Slack integration for team notifications:

### Available Slack Tools

1. **send_drift_report_to_slack()** - Send comprehensive drift reports
2. **send_drift_alert_to_slack()** - Send specific drift alerts
3. **send_drift_summary_to_slack()** - Send drift summary reports

### Setting Up Slack Webhook

1. Go to your Slack workspace
2. Create a new app or use an existing one
3. Enable "Incoming Webhooks"
4. Create a webhook for your desired channel
5. Copy the webhook URL to your `.env` file as `SLACK_WEBHOOK_URL`

### Testing Slack Integration

Run the included test script:

```bash
python test_slack_integration.py
```

## 🤖 Chatbot Capabilities

### DriftGuard Assistant

The drift-aware chatbot can:

- **Monitor Drift Status**: Check current configuration drift
- **Analyze Drift Details**: Provide detailed analysis of drift incidents
- **Trigger Analysis**: Manually trigger drift detection
- **Send Notifications**: Automatically send alerts to Slack
- **Provide Recommendations**: Suggest actions for drift resolution

### Example Interactions

**User:** "What's the current drift status?"
**Assistant:** *Checks DriftGuard APIs and provides real-time status*

**User:** "Send a drift alert to Slack"
**Assistant:** *Sends formatted alert with current drift information*

**User:** "Explain what configuration drift means"
**Assistant:** *Provides educational context about GitOps and drift*

## 🧪 Testing

### Test Slack Integration

```bash
python test_slack_integration.py
```

### Test DriftGuard Integration

```bash
# Ensure DriftGuard backend is running
curl http://localhost:8080/health

# Test chatbot integration
curl -X POST "http://localhost:8000/drift-query" \
  -H "Content-Type: application/json" \
  -d '{"topic": "Check drift health"}'
```

## 📁 Project Structure

```
smartserve-chatbot/
├── src/
│   ├── tools/
│   │   ├── driftguard_tool.py    # DriftGuard API integration
│   │   └── slack_tool.py         # Slack notification tools
│   ├── nodes/
│   │   └── drift_aware_chatbot_node.py  # AI chatbot logic
│   ├── llm/
│   │   └── openai_llm.py         # OpenAI integration
│   └── graphs/
│       └── graph_builder.py      # LangGraph workflow
├── app.py                        # FastAPI application
├── test_slack_integration.py     # Slack testing script
├── requirements.txt              # Python dependencies
└── config.example.env           # Environment configuration
```

## 🔗 Integration with DriftGuard

This chatbot is designed to work seamlessly with the DriftGuard backend:

- **Real-time Monitoring**: Connects to DriftGuard APIs for live data
- **Automated Alerts**: Sends notifications when drift is detected
- **Team Communication**: Keeps teams informed through Slack
- **Intelligent Analysis**: Provides AI-powered insights and recommendations

## 📝 License

This project is part of the DriftGuard ecosystem. See the main project LICENSE for details.
