package app

import "net/http"

// response returns the canned Manus API payload for a request path.
func (f *fakeManus) response(r *http.Request, baseURL string) interface{} {
	switch r.URL.Path {
	case "/v2/task.create":
		return map[string]interface{}{
			"ok": true, "request_id": "req_create", "task_id": "task_1",
			"task_title": "Demo task", "task_url": "https://manus.im/app/task_1",
		}
	case "/v2/task.list":
		return map[string]interface{}{
			"ok": true, "request_id": "req_list", "has_more": false,
			"data": []map[string]interface{}{{
				"id": "task_1", "title": "Demo task", "status": "running",
				"share_visibility": "private", "credit_usage": 1.5,
				"created_at": 1700000000000, "updated_at": 1700000001000,
			}},
		}
	case "/v2/task.detail":
		return map[string]interface{}{
			"ok": true, "request_id": "req_detail",
			"task": map[string]interface{}{
				"id": "task_1", "title": "Demo task", "status": "finished",
				"share_visibility": "private", "credit_usage": 2,
				"created_at": 1700000000000, "updated_at": 1700000002000,
				"task_url": "https://manus.im/app/task_1",
			},
		}
	case "/v2/task.update":
		// The SDK decodes the flat update response (task_id/task_title/...).
		return map[string]interface{}{
			"ok": true, "request_id": "req_update", "task_id": "task_1",
			"task_title": "Renamed", "task_url": "https://manus.im/app/task_1",
			"share_visibility": "private",
		}
	case "/v2/task.delete":
		return map[string]interface{}{"ok": true, "request_id": "req_delete", "deleted": true}
	case "/v2/task.listMessages":
		return map[string]interface{}{
			"ok": true, "request_id": "req_messages", "task_id": "task_1", "has_more": false,
			"messages": []map[string]interface{}{
				{
					"id": "m1", "type": "user_message", "timestamp": 1700000000000,
					"user_message": map[string]interface{}{
						"content": []map[string]interface{}{{"type": "text", "text": "hello"}},
					},
				},
				{
					"id": "m2", "type": "assistant_message", "timestamp": "2024-01-01T00:00:02Z",
					"assistant_message": map[string]interface{}{
						"content": []map[string]interface{}{{"type": "text", "text": "hi there"}},
					},
				},
			},
		}
	case "/v2/task.sendMessage":
		return map[string]interface{}{"ok": true, "request_id": "req_send", "task_id": "task_1"}
	case "/v2/task.stop":
		return map[string]interface{}{"ok": true, "request_id": "req_stop"}
	case "/v2/task.confirmAction":
		return map[string]interface{}{"ok": true, "request_id": "req_confirm", "task_id": "task_1", "confirmed": true}
	case "/v2/file.upload":
		return map[string]interface{}{
			"ok": true, "request_id": "req_file",
			"file":              map[string]interface{}{"id": "file_1", "filename": "note.txt", "status": "created"},
			"upload_url":        baseURL + "/upload/file_1",
			"upload_expires_at": 1700000900000,
		}
	case "/v2/file.detail":
		return map[string]interface{}{
			"ok": true, "request_id": "req_file_detail",
			"file": map[string]interface{}{
				"id": "file_1", "filename": "note.txt", "status": "uploaded",
				"size_bytes": 11, "created_at": 1700000000000, "expires_at": 1700000900000,
			},
		}
	case "/v2/file.delete":
		return map[string]interface{}{"ok": true, "request_id": "req_file_delete", "deleted": true}
	case "/v2/usage.list":
		return map[string]interface{}{
			"ok": true, "request_id": "req_usage", "has_more": false,
			"data": []map[string]interface{}{{
				"task_id": "task_1", "title": "Demo", "credits": 2, "created_at": 1700000000000,
			}},
		}
	case "/v2/usage.availableCredits":
		return map[string]interface{}{
			"ok": true, "request_id": "req_credits",
			"data": map[string]interface{}{"total_credits": 100, "free_credits": 50, "periodic_credits": 50},
		}
	case "/v2/agent.list":
		return map[string]interface{}{"ok": true, "request_id": "req_agents", "data": []map[string]interface{}{{"id": "agent_1", "nickname": "Scout"}}}
	case "/v2/project.list":
		return map[string]interface{}{"ok": true, "request_id": "req_projects", "data": []map[string]interface{}{{"id": "project_1", "name": "Research"}}}
	case "/v2/project.create":
		return map[string]interface{}{"ok": true, "request_id": "req_project", "project": map[string]interface{}{"id": "project_2", "name": "New"}}
	case "/v2/skill.list":
		return map[string]interface{}{"ok": true, "request_id": "req_skills", "data": []map[string]interface{}{{"id": "skill_1", "name": "Browser"}}}
	case "/v2/connector.list":
		return map[string]interface{}{"ok": true, "request_id": "req_connectors", "data": []map[string]interface{}{{"id": "conn_1", "name": "Gmail"}}}
	case "/v2/browser.onlineList":
		return map[string]interface{}{"ok": true, "request_id": "req_browsers", "data": []map[string]interface{}{{"client_id": "b1", "client_name": "Laptop"}}}
	case "/v2/webhook.list":
		return map[string]interface{}{"ok": true, "request_id": "req_webhooks", "data": []map[string]interface{}{{"id": "wh_1", "url": "https://example.com/hook"}}}
	case "/v2/webhook.create":
		return map[string]interface{}{"ok": true, "request_id": "req_webhook", "webhook": map[string]interface{}{"id": "wh_2"}}
	case "/v2/webhook.delete", "/upload/file_1":
		return map[string]interface{}{"ok": true}
	default:
		return map[string]interface{}{"ok": false, "error": "unexpected path " + r.URL.Path}
	}
}
