import { useState, useEffect } from 'react';
import { fetchTodos, addTodo, updateTodo, deleteTodo } from '../../../api/projects';
import type { Todo } from '../../../api/projects';
import './ProjectTodos.css';

interface ProjectTodosProps {
    projectId: string;
}

export function ProjectTodos({ projectId }: ProjectTodosProps) {
    const [todos, setTodos] = useState<Todo[]>([]);
    const [loading, setLoading] = useState(true);
    const [newTodoText, setNewTodoText] = useState('');
    const [editingId, setEditingId] = useState<string | null>(null);
    const [editText, setEditText] = useState('');
    const [error, setError] = useState('');

    useEffect(() => {
        loadTodos();
    }, [projectId]);

    const loadTodos = async () => {
        try {
            setLoading(true);
            const data = await fetchTodos(projectId);
            setTodos(data);
        } catch (err) {
            console.error('Failed to load todos:', err);
        } finally {
            setLoading(false);
        }
    };

    const handleAdd = async () => {
        if (!newTodoText.trim()) return;
        try {
            setNewTodoText('');
            const newTodo = await addTodo(projectId, newTodoText.trim());
            setTodos([...todos, newTodo]);
            setError('');
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to add todo');
        }
    };

    const handleToggle = async (todo: Todo) => {
        try {
            const updated = await updateTodo(projectId, todo.id, { done: !todo.done });
            setTodos(todos.map(t => t.id === todo.id ? updated : t));
            setError('');
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to update todo');
        }
    };

    const handleEdit = (todo: Todo) => {
        setEditingId(todo.id);
        setEditText(todo.text);
    };

    const handleSaveEdit = async (id: string) => {
        if (!editText.trim()) {
            setEditingId(null);
            return;
        }
        try {
            const updated = await updateTodo(projectId, id, { text: editText.trim() });
            setTodos(todos.map(t => t.id === id ? updated : t));
            setEditingId(null);
            setError('');
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to update todo');
        }
    };

    const handleCancelEdit = () => {
        setEditingId(null);
        setEditText('');
    };

    const handleDelete = async (id: string) => {
        if (!confirm('Are you sure you want to delete this todo?')) return;
        try {
            await deleteTodo(projectId, id);
            setTodos(todos.filter(t => t.id !== id));
            setError('');
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to delete todo');
        }
    };

    return (
        <div style={{ padding: '16px', marginTop: 16 }}>
            <div style={{ fontSize: '15px', fontWeight: 600, color: '#e2e8f0', marginBottom: 12 }}>
                Project TODOs
            </div>

            {error && (
                <div className="mcc-todo-error">
                    {error}
                </div>
            )}

            <div className="mcc-todo-add">
                <input
                    type="text"
                    className="mcc-todo-input"
                    placeholder="Add a new task..."
                    value={newTodoText}
                    onChange={(e) => setNewTodoText(e.target.value)}
                    onKeyPress={(e) => e.key === 'Enter' && handleAdd()}
                />
                <button
                    className="mcc-todo-btn"
                    onClick={handleAdd}
                    disabled={!newTodoText.trim()}
                >
                    Add
                </button>
            </div>

            {loading ? (
                <div className="mcc-todo-empty">Loading...</div>
            ) : todos.length === 0 ? (
                <div className="mcc-todo-empty">No tasks yet. Add one above to get started.</div>
            ) : (
                <div className="mcc-todo-list">
                    {todos.map(todo => (
                        <div key={todo.id} className={`mcc-todo-item ${todo.done ? 'done' : ''}`}>
                            <input
                                type="checkbox"
                                className="mcc-todo-checkbox"
                                checked={todo.done}
                                onChange={() => handleToggle(todo)}
                            />

                            {editingId === todo.id ? (
                                <input
                                    type="text"
                                    className="mcc-todo-edit-input"
                                    value={editText}
                                    onChange={(e) => setEditText(e.target.value)}
                                    onKeyPress={(e) => e.key === 'Enter' && handleSaveEdit(todo.id)}
                                    autoFocus
                                />
                            ) : (
                                <span className="mcc-todo-text">{todo.text}</span>
                            )}

                            <div className="mcc-todo-actions">
                                {editingId === todo.id ? (
                                    <>
                                        <button
                                            className="mcc-todo-icon-btn"
                                            onClick={() => handleSaveEdit(todo.id)}
                                        >
                                            ✓
                                        </button>
                                        <button
                                            className="mcc-todo-icon-btn"
                                            onClick={handleCancelEdit}
                                        >
                                            ✕
                                        </button>
                                    </>
                                ) : (
                                    <>
                                        <button
                                            className="mcc-todo-icon-btn"
                                            onClick={() => handleEdit(todo)}
                                        >
                                            ✎
                                        </button>
                                        <button
                                            className="mcc-todo-icon-btn danger"
                                            onClick={() => handleDelete(todo.id)}
                                        >
                                            🗑
                                        </button>
                                    </>
                                )}
                            </div>
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
}
