import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import DefectDetailPage from './DefectDetailPage';
import { fetchAssignableUsers, fetchDefect, fetchDefectFlowLogs, updateDefect } from '../api';
import { useAuth } from '../context/AuthContext';

jest.mock('../api', () => ({
  fetchAssignableUsers: jest.fn(), fetchDefect: jest.fn(), fetchDefectFlowLogs: jest.fn(),
  updateDefect: jest.fn(), getImageUrl: jest.fn(), downloadFile: jest.fn(),
  executeDefectAction: jest.fn(), exportDefectDetailPdf: jest.fn(),
}));
jest.mock('../context/AuthContext', () => ({ useAuth: jest.fn() }));
jest.mock('react-router-dom', () => ({
  useNavigate: () => jest.fn(),
  useParams: () => ({ id: '42' }),
  useSearchParams: () => [new URLSearchParams('edit=1'), jest.fn()],
}));

beforeAll(() => {
  global.ResizeObserver = class { observe() {} unobserve() {} disconnect() {} };
  // JSDOM does not implement MessageChannel; preserve asynchronous delivery.
  global.MessageChannel = class {
    constructor() {
      this.port1 = { onmessage: null, close() {} };
      this.port2 = { postMessage: (data) => setTimeout(() => this.port1.onmessage?.({ data }), 0), close() {} };
    }
  };
  window.matchMedia = () => ({
    matches: false, addListener: jest.fn(), removeListener: jest.fn(),
    addEventListener: jest.fn(), removeEventListener: jest.fn(), dispatchEvent: jest.fn(),
  });
  window.getComputedStyle = ((original) => (element) => original(element))(window.getComputedStyle);
});
beforeEach(() => {
  jest.clearAllMocks();
  useAuth.mockReturnValue({ currentUser: { username: 'inspector' }, hasRole: () => true });
  fetchDefect.mockResolvedValue({ id: 42, name: '原始标题', severity: 'high', status: 'processing', description: '原始描述', available_actions: [] });
  fetchDefectFlowLogs.mockResolvedValue([]);
  fetchAssignableUsers.mockResolvedValue([]);
  updateDefect.mockResolvedValue({});
});

test('editor saves basic fields without modifying workflow status', async () => {
  render(<DefectDetailPage />);
  const title = await screen.findByLabelText('标题');
  await waitFor(() => expect(title.value).toBe('原始标题'));
  fireEvent.change(title, { target: { value: '更新标题' } });
  fireEvent.change(screen.getByLabelText('描述'), { target: { value: '更新描述' } });
  fireEvent.click(screen.getByRole('button', { name: 'OK' }));
  await waitFor(() => expect(updateDefect).toHaveBeenCalledWith('42', { name: '更新标题', severity: 'high', description: '更新描述' }));
});

test('viewer cannot open editor through query string', async () => {
  useAuth.mockReturnValue({ currentUser: { username: 'viewer' }, hasRole: () => false });
  render(<DefectDetailPage />);
  await screen.findByRole('heading', { name: '原始标题' });
  expect(screen.queryByRole('dialog')).toBeNull();
  expect(screen.queryByRole('button', { name: '编辑信息' })).toBeNull();
});
