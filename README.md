# 🧠 AI Helper Server

A backend server built in Go for managing tasks and chatting with an AI assistant powered by **Google Gemini**. It supports:

- 💬 Context-aware AI conversations
- ✅ Task creation, updating, deleting
- � Smart session handling and summarization
- 🔐 Supabase integration for storage and authentication

---

## 📦 Setup

### 1. Clone the repo

```bash
git clone https://github.com/yourusername/ai-helper
cd ai-helper
```

### 2. Install dependencies

Uses Go modules.

```bash
go mod tidy
```

### 3. Environment variables

Create a `.env` file in the root:

```env
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_SERVICE_KEY=your-service-role-key
GEMINI_API_KEY=your-gemini-api-key
```

---

## 🚀 Run the server

```bash
go run main.go
```

Server runs at `http://localhost:8080`

---

## 🔐 Authentication

All routes require an Authorization header:

```
Authorization: Bearer <supabase_user_token>
```

This is parsed to identify the authenticated Supabase user.

---

## 💬 AI Chat Assistant

`POST /chat`

Chat with your AI assistant. It understands context, supports meaningful discussion, and suggests tasks when helpful.

**Request Body:**

```json
{
  "message": "I'm feeling overwhelmed by my to-do list.",
  "session_id": "optional-session-id"
}
```

If `session_id` is not provided, a new session will be created.

**Response Example:**

```json
{
  "success": true,
  "user_message": "I'm feeling overwhelmed by my to-do list.",
  "ai_response": "Let's break this down into smaller tasks to make it manageable.",
  "action_items": [
    {
      "title": "List top 3 urgent items",
      "description": "Write down the 3 tasks that are most time-sensitive"
    }
  ],
  "session_id": "abc123"
}
```

**Features:**
- 🧠 Prompting strategy chooses between discussion, action items, or both.
- ✅ Suggested tasks are saved automatically to Supabase.
- 📝 Past messages and session summaries are used for context.
- 🔄 Summaries are refreshed using Gemini Pro.

---

## ✅ Task Management Endpoints

### `POST /tasks/create`

Create a task.

```json
{
  "user_id": "uuid",
  "title": "Finish wireframes",
  "description": "UI wireframes for onboarding screen",
  "status": "pending"
}
```

### `PATCH /tasks/update?id=task_id`

Update fields on a task. Request body is a JSON object of fields to update.

```json
{
  "status": "completed"
}
```

### `DELETE /tasks/delete?id=task_id`

Deletes a task belonging to the authenticated user.

### `GET /tasks`

Fetch tasks for the user.

**Query Params:**
- `session_id` (optional): filter by session
- `status` (optional): pending, completed, etc.
- `search` (optional): search title or description
- `limit` (optional): default 20
- `offset` (optional): for pagination

**Example:**

```
GET /tasks?status=pending&limit=10&offset=0
```

**Response:**

```json
{
  "success": true,
  "tasks": [ ... ],
  "limit": 10,
  "offset": 0,
  "total": 42
}
```

---

## 🧠 Prompting Philosophy

The AI uses a carefully structured prompt system that:
- Supports people emotionally and practically
- Chooses between advice, discussion, or action items
- Avoids robotic, overly task-driven replies
- Adapts to feedback during the conversation
- Stores and uses summaries of each session for future context

---