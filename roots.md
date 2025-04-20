## Prompts for Managing Roots in a Kafka‑MCP‑Server  

Roots let clients define which URIs the server should focus on—filesystem paths, API endpoints, configuration directories, and more. Below are useful prompts (slash‑commands or natural‑language) that you can expose in your Kafka‑MCP‑Server to let users and LLM agents manage roots interactively.

### 1. Slash‑Command Prompts  
- `/kafka add-root <uri> name="<display-name>"`  
  “Register a new root at `<uri>` with the name `<display-name>`.”  
- `/kafka list-roots`  
  “Show all currently registered roots (URI and name).”  
- `/kafka update-root <uri> name="<new-name>"`  
  “Change the display name of the root at `<uri>` to `<new-name>`.”  
- `/kafka remove-root <uri>`  
  “Unregister the root located at `<uri>`.”  

### 2. Natural‑Language Prompts  
- “Add a root for my local config directory: `file:///home/user/kafka/config` named ‘Broker Configs’.”  
- “List all roots you’re using right now.”  
- “Remove the API endpoint root `https://api.example.com/v1`.”  
- “Rename the root `file:///var/logs` to ‘Kafka Logs’.”  

### 3. Batch & Initialization Prompts  
- “Initialize roots with:  
   • `file:///home/user/projects/kafka` as ‘Project Repo’  
   • `https://metrics.example.com/api` as ‘Metrics API’”  
- “Reset roots to only include my current workspace folder.”  

### 4. Validation & Discovery Prompts  
- “Validate that all registered roots are accessible.”  
- “Suggest roots based on the current working directory.”  
- “Which roots contain configuration files?”  

### 5. Examples in JSON Format  
Expose these templates for clients that work with JSON payloads:  
```json
{
  "roots": [
    { "uri": "file:///home/user/kafka/config", "name": "Broker Configs" },
    { "uri": "https://api.monitoring/v1",   "name": "Monitoring API" }
  ]
}
```  

### Best Practices  
- Use clear, descriptive **display names** to help LLMs and users understand purpose.  
- Encourage **URI validation** to alert on unreachable roots.  
- Allow **batch registration** of multiple roots for large workspaces.  
- Support **dynamic updates** so roots can change as projects evolve.

These prompts empower conversational and programmatic control over which data sources and endpoints your Kafka‑MCP‑Server will surface to LLMs, ensuring context‑aware operations within defined boundaries.
