require 'json'
require 'net/http'
require 'uri'
require 'logger'

module TaskRunner
  class Configuration
    attr_accessor :max_workers, :retry_count, :retry_delay,
                  :timeout, :log_level, :queue_name

    def initialize
      @max_workers = 4
      @retry_count = 3
      @retry_delay = 5
      @timeout = 30
      @log_level = :info
      @queue_name = 'default'
    end

    def validate!
      raise ArgumentError, 'max_workers must be positive' unless max_workers.positive?
      raise ArgumentError, 'retry_count must be non-negative' unless retry_count >= 0
      raise ArgumentError, 'timeout must be positive' unless timeout.positive?
    end
  end

  class Task
    attr_reader :id, :name, :payload, :status, :attempts, :created_at, :error

    def initialize(name:, payload: {})
      @id = SecureRandom.uuid
      @name = name
      @payload = payload
      @status = :pending
      @attempts = 0
      @created_at = Time.now
      @error = nil
    end

    def pending?
      status == :pending
    end

    def running?
      status == :running
    end

    def completed?
      status == :completed
    end

    def failed?
      status == :failed
    end

    def run!
      @status = :running
      @attempts += 1
    end

    def complete!
      @status = :completed
    end

    def fail!(error)
      @error = error
      @status = :failed
    end

    def retry?
      failed? && attempts < 3
    end

    def reset!
      @status = :pending
      @error = nil
    end

    def to_h
      {
        id: id,
        name: name,
        payload: payload,
        status: status,
        attempts: attempts,
        created_at: created_at.iso8601,
        error: error&.message
      }
    end
  end

  class Worker
    attr_reader :id, :busy

    def initialize(id, logger:)
      @id = id
      @logger = logger
      @busy = false
      @current_task = nil
    end

    def available?
      !busy
    end

    def execute(task, handlers)
      @busy = true
      @current_task = task
      task.run!

      handler = handlers[task.name]
      unless handler
        task.fail!(RuntimeError.new("No handler for task: #{task.name}"))
        @logger.error("Worker #{id}: no handler for #{task.name}")
        return
      end

      begin
        @logger.info("Worker #{id}: executing #{task.name} (attempt #{task.attempts})")
        result = handler.call(task.payload)
        task.complete!
        @logger.info("Worker #{id}: completed #{task.name}")
        result
      rescue StandardError => e
        task.fail!(e)
        @logger.error("Worker #{id}: failed #{task.name} - #{e.message}")
        nil
      ensure
        @busy = false
        @current_task = nil
      end
    end
  end

  class Queue
    def initialize
      @tasks = []
      @mutex = Mutex.new
    end

    def push(task)
      @mutex.synchronize { @tasks.push(task) }
    end

    def pop
      @mutex.synchronize { @tasks.shift }
    end

    def size
      @mutex.synchronize { @tasks.size }
    end

    def empty?
      @mutex.synchronize { @tasks.empty? }
    end

    def pending_tasks
      @mutex.synchronize { @tasks.select(&:pending?) }
    end
  end

  class Runner
    attr_reader :config, :stats

    def initialize(config = Configuration.new)
      config.validate!
      @config = config
      @logger = Logger.new($stdout, level: config.log_level)
      @queue = Queue.new
      @workers = Array.new(config.max_workers) { |i| Worker.new(i, logger: @logger) }
      @handlers = {}
      @stats = { completed: 0, failed: 0, retried: 0 }
      @running = false
    end

    def register(task_name, &handler)
      @handlers[task_name.to_s] = handler
      @logger.info("Registered handler for: #{task_name}")
    end

    def enqueue(name, payload = {})
      task = Task.new(name: name.to_s, payload: payload)
      @queue.push(task)
      @logger.debug("Enqueued task: #{task.id} (#{name})")
      task
    end

    def start
      @running = true
      @logger.info("Starting runner with #{config.max_workers} workers")

      while @running
        break if @queue.empty? && @workers.all?(&:available?)

        worker = @workers.find(&:available?)
        if worker && !@queue.empty?
          task = @queue.pop
          next unless task

          Thread.new do
            worker.execute(task, @handlers)
            handle_result(task)
          end
        else
          sleep 0.1
        end
      end

      @logger.info("Runner stopped. Stats: #{stats}")
    end

    def stop
      @running = false
    end

    private

    def handle_result(task)
      if task.completed?
        @stats[:completed] += 1
      elsif task.retry?
        @stats[:retried] += 1
        task.reset!
        sleep config.retry_delay
        @queue.push(task)
        @logger.info("Retrying task: #{task.id} (attempt #{task.attempts + 1})")
      else
        @stats[:failed] += 1
        @logger.error("Task permanently failed: #{task.id} - #{task.error&.message}")
      end
    end
  end

  class WebhookNotifier
    def initialize(url, logger: Logger.new($stdout))
      @url = URI.parse(url)
      @logger = logger
    end

    def notify(event, data = {})
      payload = {
        event: event,
        timestamp: Time.now.iso8601,
        data: data
      }.to_json

      request = Net::HTTP::Post.new(@url.path)
      request['Content-Type'] = 'application/json'
      request.body = payload

      response = Net::HTTP.start(@url.host, @url.port) do |http|
        http.request(request)
      end

      unless response.is_a?(Net::HTTPSuccess)
        @logger.error("Webhook failed: #{response.code} #{response.message}")
      end
    rescue StandardError => e
      @logger.error("Webhook error: #{e.message}")
    end
  end
end
