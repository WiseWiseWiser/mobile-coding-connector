import { useState, useEffect } from 'react';
import { fetchTodos, addTodo, updateTodo, deleteTodo } from '../../../api/projects';
import type { Todo } from '../../../api/projects';
import './ProjectTodos.css';

interface ProjectTodosProps {
    projectId: string;
}

export function ProjectTodos({ projectId }: ProjectTodosProps) {
    console.log('[ProjectTodos] Initializing with projectId:', projectId);

    const [todos, setTodos] = useState<Todo[]>([]);
    const [loading, setLoading] = useState(true);
    const [newTodoText, setNewTodoText] = useState('');
    const [editingId, setEditingId] = useState<string | null>(null);
    const [editText, setEditText] = useState('');
    const [error, setError] = useState('');

    useEffect(() => {
        console.log('[ProjectTodos] useEffect triggered, loading todos...');
        loadTodos();
    }, [projectId]);

    const loadTodos = async () => {
        console.log('[ProjectTodos] Loading todos for project:', projectId);
        try {
            setLoading(true);
            const data = await fetchTodos(projectId);
            console.log('[ProjectTodos] Loaded todos:', data);
            setTodos(data || []);
        } catch (err) {
            console.error('[ProjectTodos] Failed to load todos:', err);
            setTodos([]);
        } finally {
            setLoading(false);
        }
    };

    const handleAdd = async () => {
        console.log('[ProjectTodos] Adding todo:', newTodoText);
        try {
            setError('');
            setNewTodoText('');
            const newTodo = await addTodo(projectId, newTodoText.trim());
            console.log('[ProjectTodos] Added todo successfully:', newTodo);
            setTodos([...todos, newTodo]);
        } catch (err) {
            console.error('[ProjectTodos] Failed to add todo:', err);
            setError(err instanceof Error ? err.message : 'Failed to add todo');
            setNewTodoText(newTodoText);
        }
    };

    const handleToggle = async (todo: Todo) => {
        console.log('[ProjectTodos] Toggling todo:', todo.id, 'done:', !todo.done);
        try {
            setError('');
            const updated = await updateTodo(projectId, todo.id, { done: !todo.done });
            console.log('[ProjectTodos] Updated todo successfully:', updated);
            setTodos(todos.map(t => t.id === todo.id ? updated : t));
        } catch (err) {
            console.error('[ProjectTodos] Failed to update todo:', err);
            setError(err instanceof Error ? err.message : 'Failed to update todo');
        }
    };

    const handleEdit = (todo: Todo) => {
        console.log('[ProjectTodos] Editing todo:', todo.id);
        setEditingId(todo.id);
        setEditText(todo.text);
    };

    const handleSaveEdit = async (id: string) => {
        if (!editText.trim()) {
            setEditingId(null);
            return;
        }
        console.log('[ProjectTodos] Saving todo edit:', id, 'text:', editText);
        try {
            setError('');
            const updated = await updateTodo(projectId, id, { text: editText.trim() });
            console.log('[ProjectTodos] Saved todo edit successfully:', updated);
            setTodos(todos.map(t => t.id === id ? updated : t));
            setEditingId(null);
        } catch (err) {
            console.error('[ProjectTodos] Failed to update todo:', err);
            setError(err instanceof Error ? err.message : 'Failed to update todo');
        }
    };

    const handleCancelEdit = () => {
        console.log('[ProjectTodos] Cancelling edit');
        setEditingId(null);
        setEditText('');
    };

    const handleDelete = async (id: string) => {
        if (!confirm('Are you sure you want to delete this todo?')) return;
        console.log('[ProjectTodos] Deleting todo:', id);
        try {
            setError('');
            await deleteTodo(projectId, id);
            console.log('[ProjectTodos] Deleted todo successfully');
            setTodos(todos.filter(t => t.id !== id));
        } catch (err) {
            console.error('[ProjectTodos] Failed to delete todo:', err);
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
