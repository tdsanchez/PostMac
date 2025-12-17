#!/usr/bin/env python3
"""
Media Server Load Testing Script

Simulates heavy concurrent load to validate architectural improvements:
- Template serialization bottleneck elimination
- Containerization readiness
- API endpoint performance
- Random mode navigation under load

Usage:
    python3 load_test.py --url http://localhost:8080 --workers 5 --duration 60
    python3 load_test.py --url http://localhost:8080 --scenario stress --workers 10
    python3 load_test.py --url http://localhost:8080 --scenario api-only --requests 1000
"""

import argparse
import concurrent.futures
import json
import random
import sys
import time
from collections import defaultdict
from dataclasses import dataclass, field
from datetime import datetime, timedelta
from typing import Dict, List, Optional
from urllib.parse import quote

import requests


@dataclass
class RequestStats:
    """Statistics for a single request"""
    endpoint: str
    duration_ms: float
    status_code: int
    error: Optional[str] = None
    timestamp: float = field(default_factory=time.time)


@dataclass
class TestResults:
    """Aggregate test results"""
    total_requests: int = 0
    successful_requests: int = 0
    failed_requests: int = 0
    broken_pipes: int = 0
    timeouts: int = 0
    total_duration_ms: float = 0.0
    min_duration_ms: float = float('inf')
    max_duration_ms: float = 0.0
    errors_by_type: Dict[str, int] = field(default_factory=lambda: defaultdict(int))
    requests_by_endpoint: Dict[str, int] = field(default_factory=lambda: defaultdict(int))
    stats: List[RequestStats] = field(default_factory=list)

    def add_request(self, stat: RequestStats):
        """Add a request statistic"""
        self.stats.append(stat)
        self.total_requests += 1
        self.requests_by_endpoint[stat.endpoint] += 1

        if stat.error:
            self.failed_requests += 1
            self.errors_by_type[stat.error] += 1
            if 'broken pipe' in stat.error.lower():
                self.broken_pipes += 1
            if 'timeout' in stat.error.lower():
                self.timeouts += 1
        else:
            self.successful_requests += 1
            self.total_duration_ms += stat.duration_ms
            self.min_duration_ms = min(self.min_duration_ms, stat.duration_ms)
            self.max_duration_ms = max(self.max_duration_ms, stat.duration_ms)

    def avg_duration_ms(self) -> float:
        """Calculate average duration"""
        if self.successful_requests == 0:
            return 0.0
        return self.total_duration_ms / self.successful_requests

    def success_rate(self) -> float:
        """Calculate success rate percentage"""
        if self.total_requests == 0:
            return 0.0
        return (self.successful_requests / self.total_requests) * 100

    def requests_per_second(self, elapsed_seconds: float) -> float:
        """Calculate throughput"""
        if elapsed_seconds == 0:
            return 0.0
        return self.total_requests / elapsed_seconds


class LoadTester:
    """Load testing orchestrator"""

    def __init__(self, base_url: str, timeout: int = 30):
        self.base_url = base_url.rstrip('/')
        self.timeout = timeout
        self.session = requests.Session()
        self.categories: List[str] = []
        self.files_by_category: Dict[str, List[str]] = {}

    def discover_categories(self) -> List[str]:
        """Fetch available categories from server"""
        try:
            response = self.session.get(f"{self.base_url}/", timeout=self.timeout)
            # Parse homepage HTML to extract category names
            # For now, use common test categories
            self.categories = ['All', '5-★★★★★', '📁 Classico']
            print(f"✓ Using test categories: {self.categories}")
            return self.categories
        except Exception as e:
            print(f"⚠ Failed to discover categories: {e}")
            self.categories = ['All']
            return self.categories

    def fetch_file_list(self, category: str) -> List[str]:
        """Fetch file list for a category via API"""
        try:
            encoded_category = quote(category)
            url = f"{self.base_url}/api/filelist?category={encoded_category}"
            response = self.session.get(url, timeout=self.timeout)

            if response.status_code == 200:
                files = response.json()
                self.files_by_category[category] = files
                print(f"✓ Fetched {len(files)} files for category '{category}'")
                return files
            else:
                print(f"⚠ Failed to fetch files for '{category}': {response.status_code}")
                return []
        except Exception as e:
            print(f"⚠ Error fetching file list for '{category}': {e}")
            return []

    def make_request(self, endpoint: str, params: Dict = None) -> RequestStats:
        """Make a single HTTP request and record statistics"""
        start = time.time()
        url = f"{self.base_url}{endpoint}"

        try:
            response = self.session.get(url, params=params, timeout=self.timeout)
            duration_ms = (time.time() - start) * 1000

            return RequestStats(
                endpoint=endpoint,
                duration_ms=duration_ms,
                status_code=response.status_code,
                error=None if response.status_code < 400 else f"HTTP {response.status_code}"
            )
        except requests.exceptions.Timeout:
            duration_ms = (time.time() - start) * 1000
            return RequestStats(
                endpoint=endpoint,
                duration_ms=duration_ms,
                status_code=0,
                error="Timeout"
            )
        except requests.exceptions.ConnectionError as e:
            duration_ms = (time.time() - start) * 1000
            error_msg = "Broken pipe" if "Broken pipe" in str(e) else "Connection error"
            return RequestStats(
                endpoint=endpoint,
                duration_ms=duration_ms,
                status_code=0,
                error=error_msg
            )
        except Exception as e:
            duration_ms = (time.time() - start) * 1000
            return RequestStats(
                endpoint=endpoint,
                duration_ms=duration_ms,
                status_code=0,
                error=str(e)
            )

    def viewer_request(self, category: str, random_mode: bool = False) -> RequestStats:
        """Make a viewer page request"""
        files = self.files_by_category.get(category, [])
        if not files:
            # Fallback - just request the category viewer without specific file
            return self.make_request(f"/view/{quote(category)}")

        # Pick a random file from the category
        file_path = random.choice(files)
        params = {'file': file_path}
        if random_mode:
            params['random'] = 'true'

        return self.make_request(f"/view/{quote(category)}", params)

    def gallery_request(self, category: str, page: int = 1) -> RequestStats:
        """Make a gallery page request"""
        return self.make_request(f"/tag/{quote(category)}", {'page': page})

    def api_filelist_request(self, category: str) -> RequestStats:
        """Make an API file list request"""
        return self.make_request(f"/api/filelist", {'category': category})

    def homepage_request(self) -> RequestStats:
        """Make a homepage request"""
        return self.make_request("/")

    def mixed_workload(self) -> RequestStats:
        """Execute a random mixed workload"""
        if not self.categories:
            return self.homepage_request()

        category = random.choice(self.categories)
        workload_type = random.choices(
            ['viewer', 'gallery', 'api', 'homepage'],
            weights=[40, 30, 20, 10]  # Weighted distribution
        )[0]

        if workload_type == 'viewer':
            return self.viewer_request(category, random_mode=random.choice([True, False]))
        elif workload_type == 'gallery':
            return self.gallery_request(category, page=random.randint(1, 5))
        elif workload_type == 'api':
            return self.api_filelist_request(category)
        else:
            return self.homepage_request()


def worker_thread(tester: LoadTester, scenario: str, duration_seconds: int,
                  requests_count: Optional[int] = None) -> List[RequestStats]:
    """Worker thread that executes requests"""
    results = []
    start_time = time.time()
    request_count = 0

    while True:
        # Check termination conditions
        elapsed = time.time() - start_time
        if duration_seconds > 0 and elapsed >= duration_seconds:
            break
        if requests_count and request_count >= requests_count:
            break

        # Execute request based on scenario
        if scenario == 'mixed':
            stat = tester.mixed_workload()
        elif scenario == 'viewer-random':
            category = random.choice(tester.categories) if tester.categories else 'All'
            stat = tester.viewer_request(category, random_mode=True)
        elif scenario == 'gallery':
            category = random.choice(tester.categories) if tester.categories else 'All'
            stat = tester.gallery_request(category)
        elif scenario == 'api-only':
            category = random.choice(tester.categories) if tester.categories else 'All'
            stat = tester.api_filelist_request(category)
        elif scenario == 'stress':
            # Aggressive stress test - no delays
            stat = tester.mixed_workload()
        else:
            stat = tester.mixed_workload()

        results.append(stat)
        request_count += 1

        # Add small delay for non-stress scenarios
        if scenario != 'stress' and duration_seconds > 0:
            time.sleep(0.1)

    return results


def print_results(results: TestResults, elapsed_seconds: float):
    """Print formatted test results"""
    print("\n" + "="*80)
    print("LOAD TEST RESULTS")
    print("="*80)

    print(f"\n📊 Overall Statistics:")
    print(f"  Total Requests:      {results.total_requests:,}")
    print(f"  Successful:          {results.successful_requests:,} ({results.success_rate():.2f}%)")
    print(f"  Failed:              {results.failed_requests:,}")
    print(f"  Broken Pipes:        {results.broken_pipes:,} {'⚠️  BOTTLENECK!' if results.broken_pipes > 0 else '✅'}")
    print(f"  Timeouts:            {results.timeouts:,}")
    print(f"  Test Duration:       {elapsed_seconds:.2f}s")
    print(f"  Throughput:          {results.requests_per_second(elapsed_seconds):.2f} req/s")

    if results.successful_requests > 0:
        print(f"\n⏱️  Response Times:")
        print(f"  Average:             {results.avg_duration_ms():.2f} ms")
        print(f"  Min:                 {results.min_duration_ms:.2f} ms")
        print(f"  Max:                 {results.max_duration_ms:.2f} ms")

        # Calculate percentiles
        durations = sorted([s.duration_ms for s in results.stats if not s.error])
        if durations:
            p50 = durations[int(len(durations) * 0.50)]
            p95 = durations[int(len(durations) * 0.95)]
            p99 = durations[int(len(durations) * 0.99)]
            print(f"  p50:                 {p50:.2f} ms")
            print(f"  p95:                 {p95:.2f} ms")
            print(f"  p99:                 {p99:.2f} ms")

    if results.requests_by_endpoint:
        print(f"\n🎯 Requests by Endpoint:")
        for endpoint, count in sorted(results.requests_by_endpoint.items(), key=lambda x: x[1], reverse=True):
            print(f"  {endpoint:30} {count:,}")

    if results.errors_by_type:
        print(f"\n❌ Errors by Type:")
        for error_type, count in sorted(results.errors_by_type.items(), key=lambda x: x[1], reverse=True):
            print(f"  {error_type:30} {count:,}")

    print("\n" + "="*80)

    # Verdict
    if results.broken_pipes > 0:
        print("⚠️  VERDICT: BROKEN PIPES DETECTED - Template serialization bottleneck still present!")
    elif results.success_rate() >= 99.0 and results.avg_duration_ms() < 500:
        print("✅ VERDICT: EXCELLENT - System handles load gracefully!")
    elif results.success_rate() >= 95.0:
        print("✓ VERDICT: GOOD - System performs adequately under load")
    else:
        print("⚠️  VERDICT: POOR - High error rate indicates issues")
    print("="*80 + "\n")


def main():
    parser = argparse.ArgumentParser(
        description="Load test the media server",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Scenarios:
  mixed          Mixed workload (default) - homepage, gallery, viewer, API
  viewer-random  Focus on viewer with random mode (stress test serialization)
  gallery        Gallery page requests with pagination
  api-only       API endpoint testing (/api/filelist)
  stress         Aggressive stress test with no delays

Examples:
  # Standard mixed load test with 5 workers for 60 seconds
  python3 load_test.py --url http://localhost:8080 --workers 5 --duration 60

  # Stress test viewer with random mode (tests serialization bottleneck)
  python3 load_test.py --url http://localhost:8080 --scenario viewer-random --workers 10 --duration 30

  # API endpoint performance test
  python3 load_test.py --url http://localhost:8080 --scenario api-only --requests 1000

  # Maximum stress test
  python3 load_test.py --url http://localhost:8080 --scenario stress --workers 20 --duration 120
        """
    )

    parser.add_argument('--url', default='http://localhost:8080',
                        help='Base URL of media server (default: http://localhost:8080)')
    parser.add_argument('--workers', type=int, default=5,
                        help='Number of concurrent workers (default: 5)')
    parser.add_argument('--duration', type=int, default=60,
                        help='Test duration in seconds (default: 60, 0 for request count only)')
    parser.add_argument('--requests', type=int, default=None,
                        help='Number of requests per worker (overrides duration)')
    parser.add_argument('--scenario', default='mixed',
                        choices=['mixed', 'viewer-random', 'gallery', 'api-only', 'stress'],
                        help='Test scenario (default: mixed)')
    parser.add_argument('--timeout', type=int, default=30,
                        help='Request timeout in seconds (default: 30)')
    parser.add_argument('--no-warmup', action='store_true',
                        help='Skip warmup phase')

    args = parser.parse_args()

    print(f"\n{'='*80}")
    print(f"MEDIA SERVER LOAD TEST")
    print(f"{'='*80}")
    print(f"URL:       {args.url}")
    print(f"Workers:   {args.workers}")
    print(f"Duration:  {args.duration}s" + (" (or until request count)" if args.requests else ""))
    print(f"Scenario:  {args.scenario}")
    print(f"Timeout:   {args.timeout}s")
    print(f"{'='*80}\n")

    # Initialize tester
    tester = LoadTester(args.url, timeout=args.timeout)

    # Warmup phase
    if not args.no_warmup:
        print("🔥 Warmup phase...")
        tester.discover_categories()
        for category in tester.categories[:3]:  # Warmup first 3 categories
            tester.fetch_file_list(category)
        print("✓ Warmup complete\n")

    # Start load test
    print(f"🚀 Starting load test with {args.workers} workers...")
    start_time = time.time()

    # Execute load test with thread pool
    with concurrent.futures.ThreadPoolExecutor(max_workers=args.workers) as executor:
        futures = [
            executor.submit(worker_thread, tester, args.scenario, args.duration, args.requests)
            for _ in range(args.workers)
        ]

        # Wait for completion
        concurrent.futures.wait(futures)

        # Collect results
        all_results = TestResults()
        for future in futures:
            worker_stats = future.result()
            for stat in worker_stats:
                all_results.add_request(stat)

    elapsed_seconds = time.time() - start_time

    # Print results
    print_results(all_results, elapsed_seconds)

    # Exit with appropriate code
    sys.exit(0 if all_results.broken_pipes == 0 else 1)


if __name__ == '__main__':
    main()
