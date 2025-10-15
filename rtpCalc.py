import random
import time
from collections import Counter
import threading

class SlotMachineSimulator:
    def __init__(self):
        self.total_spins = 0
        self.total_wagered = 0.0
        self.total_won = 0.0
        self.bet_amount = 1.0  # Fixed bet amount for simulation
        self.running = True
        
    def spin_slot(self):
        """Simulate a single slot machine spin"""
        # Generate 3 random numbers between 0 and 11
        result = [random.randint(0, 11) for _ in range(3)]
        
        # Count symbol occurrences
        counts = Counter(result)
        
        # Calculate payout based on your paytable
        payout = 0.0
        for sym, count in counts.items():
            # Skip ❌ (index 7 in symbol array)
            if sym == 7:
                continue
                
            if count == 3:
                if sym == 0:  # 7️⃣ jackpot
                    payout = 77.7
                else:
                    payout = 33.3
                break
            elif count == 2:
                if sym == 0:  # two 7️⃣
                    payout = 7.7
                else:
                    payout = 3
                break
        
        # Calculate win amount
        if payout > 0:
            win_amount = payout * self.bet_amount
        else:
            win_amount = 0.0
            
        return win_amount
    
    def calculate_rtp(self):
        """Calculate Return to Player percentage"""
        if self.total_wagered == 0:
            return 0.0
        return (self.total_won / self.total_wagered) * 100
    
    def run_simulation(self):
        """Run the slot machine simulation"""
        print("Starting slot machine RTP simulation...")
        print("Press Ctrl+C to stop\n")
        
        while self.running:
            try:
                # Run many spins in a batch for better performance
                batch_size = 100000
                batch_won = 0.0
                
                for _ in range(batch_size):
                    win_amount = self.spin_slot()
                    batch_won += win_amount
                    self.total_spins += 1
                
                # Update totals
                self.total_wagered += batch_size * self.bet_amount
                self.total_won += batch_won
                
                # Sleep briefly to allow for interruption
                time.sleep(0.001)
                
            except KeyboardInterrupt:
                self.running = False
                break
    
    def print_stats(self):
        """Print statistics every second"""
        while self.running:
            try:
                rtp = self.calculate_rtp()
                print(f"Spins: {self.total_spins:,} | "
                      f"RTP: {rtp:.4f}% | "
                      f"Wagered: ${self.total_wagered:,.2f} | "
                      f"Won: ${self.total_won:,.2f}")
                time.sleep(1)
            except KeyboardInterrupt:
                self.running = False
                break

def main():
    simulator = SlotMachineSimulator()
    
    # Start simulation in separate thread
    sim_thread = threading.Thread(target=simulator.run_simulation)
    sim_thread.daemon = True
    sim_thread.start()
    
    # Start stats printing in main thread
    try:
        simulator.print_stats()
    except KeyboardInterrupt:
        simulator.running = False
        print("\n\nSimulation stopped by user.")
    
    # Print final statistics
    print(f"\n=== FINAL RESULTS ===")
    print(f"Total Spins: {simulator.total_spins:,}")
    print(f"Total Wagered: ${simulator.total_wagered:,.2f}")
    print(f"Total Won: ${simulator.total_won:,.2f}")
    print(f"Final RTP: {simulator.calculate_rtp():.6f}%")
    print(f"House Edge: {100 - simulator.calculate_rtp():.6f}%")

if __name__ == "__main__":
    main()
