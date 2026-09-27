import React from 'react';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import DefectListPage from './DefectListPage';
import { fetchDefects, fetchProductionLines, fetchEquipment, fetchDefectTypes } from '../api';

jest.mock('../api', () => ({ fetchDefects: jest.fn(), fetchProductionLines: jest.fn(), fetchEquipment: jest.fn(), fetchDefectTypes: jest.fn() }));
jest.mock('../context/AuthContext', () => ({ useAuth: () => ({ hasRole: () => false }) }));
jest.mock('react-router-dom', () => ({ useNavigate: () => jest.fn() }));
jest.mock('antd', () => {
  const Box = ({ children }) => <div>{children}</div>;
  return {
    Alert: ({ message, action }) => <div role="alert">{message}{action}</div>,
    Button: ({ children, onClick }) => <button onClick={onClick}>{children}</button>,
    Card: Box, Space: Box, Image: Box, Popconfirm: Box, Tag: Box,
    Typography: { Title: Box, Text: Box }, message: { error: jest.fn() },
    Select: ({ onChange, placeholder }) => <button onClick={() => onChange('high')}>{placeholder}</button>,
    Table: ({ dataSource, loading, pagination }) => <div>
      <span data-testid="loading">{String(loading)}</span>
      {dataSource.map(item => <div key={item.id}>{item.name}</div>)}
      <button onClick={() => pagination.onChange(1, 20)}>每页20条</button>
    </div>,
  };
});

beforeEach(() => {
  jest.clearAllMocks();
  [fetchProductionLines, fetchEquipment, fetchDefectTypes].forEach(fn => fn.mockResolvedValue({ items: [] }));
});

test('ignores a stale failure after a newer successful filter request', async () => {
  let rejectOld;
  fetchDefects.mockImplementationOnce(() => new Promise((resolve, reject) => { rejectOld = reject; }))
    .mockResolvedValueOnce({ items: [{ id: 2, name: '最新结果' }], total: 1 });
  render(<DefectListPage />);
  fireEvent.click(screen.getByText('按等级筛选'));
  expect(await screen.findByText('最新结果')).toBeInTheDocument();
  await act(async () => rejectOld(new Error('old request failed')));
  expect(screen.queryByText(/缺陷列表加载失败/)).not.toBeInTheDocument();
  expect(fetchDefects.mock.calls[0][1].signal.aborted).toBe(true);
  expect(screen.getByTestId('loading')).toHaveTextContent('false');
});

test('shows a retry action and recovers from a failed request', async () => {
  fetchDefects.mockRejectedValueOnce(new Error('network')).mockResolvedValueOnce({ items: [{ id: 1, name: '恢复结果' }], total: 1 });
  render(<DefectListPage />);
  fireEvent.click(await screen.findByText('重试'));
  expect(await screen.findByText('恢复结果')).toBeInTheDocument();
  expect(screen.queryByText(/缺陷列表加载失败/)).not.toBeInTheDocument();
});

test('changing page size makes only one request and unmount cancels pending work', async () => {
  fetchDefects.mockResolvedValue({ items: [], total: 0 });
  const view = render(<DefectListPage />);
  await waitFor(() => expect(screen.getByTestId('loading')).toHaveTextContent('false'));
  fireEvent.click(screen.getByText('每页20条'));
  await waitFor(() => expect(screen.getByTestId('loading')).toHaveTextContent('false'));
  expect(fetchDefects).toHaveBeenCalledTimes(2);
  view.unmount();
  expect(fetchDefects.mock.calls[1][1].signal.aborted).toBe(true);
});
