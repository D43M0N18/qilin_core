import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import userEvent from '@testing-library/user-event';

// Simple Button component for testing
const Button = ({ 
  children, 
  onClick, 
  disabled = false 
}: { 
  children: React.ReactNode; 
  onClick?: () => void; 
  disabled?: boolean;
}) => (
  <button onClick={onClick} disabled={disabled}>
    {children}
  </button>
);

describe('Button Component', () => {
  it('renders children correctly', () => {
    render(<Button>Click me</Button>);
    expect(screen.getByText('Click me')).toBeInTheDocument();
  });

  it('calls onClick when clicked', async () => {
    const handleClick = vi.fn();
    const user = userEvent.setup();
    
    render(<Button onClick={handleClick}>Click me</Button>);
    
    await user.click(screen.getByText('Click me'));
    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  it('does not call onClick when disabled', async () => {
    const handleClick = vi.fn();
    const user = userEvent.setup();
    
    render(
      <Button onClick={handleClick} disabled={true}>
        Click me
      </Button>
    );
    
    await user.click(screen.getByText('Click me'));
    expect(handleClick).not.toHaveBeenCalled();
  });
});
