package prometheus

import (
	"PrometheusF6005/ont"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"log"
	"strconv"
	"sync"
	"time"
)

// ONTCollector implements the prometheus.Collector interface
type ONTCollector struct {
	mu       sync.Mutex
	session  *ont.Session
	endpoint string
	username string
	password string
	loginFn  func(string, string, string) (*ont.Session, error)
	now      func() time.Time

	reloginDelay time.Duration
	nextLoginAt  time.Time
}

// NewONTCollector creates a new ONT metrics collector
func NewONTCollector(endpoint, username, password string, reloginDelay time.Duration) *ONTCollector {
	return &ONTCollector{
		endpoint:     endpoint,
		username:     username,
		password:     password,
		loginFn:      ont.Login,
		now:          time.Now,
		reloginDelay: reloginDelay,
	}
}

// Describe implements prometheus.Collector
func (c *ONTCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- deviceInfoDesc
	ch <- cpuUsageDesc
	ch <- memoryUsageDesc
	ch <- uptimeDesc
	ch <- bytesDesc
	ch <- packetsDesc
	ch <- errorsDesc
	ch <- discardsDesc
	ch <- networkStatusDesc
	ch <- opticalSignalDesc
	ch <- opticalTempDesc
	ch <- opticalVoltageDesc
	ch <- opticalBiasCurrentDesc
	ch <- opticalStatusDesc
	ch <- rfPowerDesc
	ch <- opticalUptimeDesc
}

func (c *ONTCollector) login() error {
	if c.now().Before(c.nextLoginAt) {
		return fmt.Errorf("waiting until %s before refreshing the ONT session", c.nextLoginAt.Format(time.RFC3339))
	}

	session, err := c.loginFn(c.endpoint, c.username, c.password)
	if err != nil {
		return err
	}
	c.session = session
	c.nextLoginAt = time.Time{}
	log.Println("Login succeeded")
	return nil
}

func loadWithRelogin[T any](c *ONTCollector, load func(*ont.Session) (*T, error)) (*T, error) {
	if c.session == nil {
		if err := c.login(); err != nil {
			return nil, err
		}
	}

	result, err := load(c.session)
	if err == nil {
		return result, nil
	}

	log.Printf("ONT request failed (%v), scheduling a new session in %s", err, c.reloginDelay)
	if c.session.Client != nil {
		c.session.CloseIdleConnections()
	}
	c.session = nil
	c.nextLoginAt = c.now().Add(c.reloginDelay)
	return nil, err
}

// Collect implements prometheus.Collector
func (c *ONTCollector) Collect(ch chan<- prometheus.Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Collect Device Info
	deviceInfo, err := loadWithRelogin(c, (*ont.Session).LoadDeviceInfo)
	if err != nil {
		log.Printf("Unable to collect device info: %v", err)
		return
	}

	ch <- prometheus.MustNewConstMetric(
		deviceInfoDesc,
		prometheus.GaugeValue,
		1,
		deviceInfo.Manufacturer,
		deviceInfo.ManufacturerOui,
		deviceInfo.VersionDate,
		deviceInfo.BootVersion,
		deviceInfo.SofwareVersion,
		deviceInfo.SoftwareVersionExtended,
		deviceInfo.SerialNumber,
		deviceInfo.Model,
		deviceInfo.HardwareVersion,
		deviceInfo.OnuAlias,
	)

	// CPU Usage metrics
	ch <- prometheus.MustNewConstMetric(
		cpuUsageDesc,
		prometheus.GaugeValue,
		float64(deviceInfo.CPUUsage1),
		"1",
	)
	ch <- prometheus.MustNewConstMetric(
		cpuUsageDesc,
		prometheus.GaugeValue,
		float64(deviceInfo.CPUUsage2),
		"2",
	)
	ch <- prometheus.MustNewConstMetric(
		cpuUsageDesc,
		prometheus.GaugeValue,
		float64(deviceInfo.CPUUsage3),
		"3",
	)
	ch <- prometheus.MustNewConstMetric(
		cpuUsageDesc,
		prometheus.GaugeValue,
		float64(deviceInfo.CPUUsage4),
		"4",
	)

	// Memory Usage metric
	ch <- prometheus.MustNewConstMetric(
		memoryUsageDesc,
		prometheus.GaugeValue,
		float64(deviceInfo.MemoryUsage),
	)

	// Uptime metric
	ch <- prometheus.MustNewConstMetric(
		uptimeDesc,
		prometheus.GaugeValue,
		float64(deviceInfo.Uptime),
	)

	// Collect LAN Info
	lanInfo, err := loadWithRelogin(c, (*ont.Session).LoadLanInfo)
	if err != nil {
		log.Printf("Unable to collect LAN info: %v", err)
		return
	}
	// Network traffic metrics
	ch <- prometheus.MustNewConstMetric(
		bytesDesc,
		prometheus.CounterValue,
		float64(lanInfo.BytesIn),
		"in",
	)
	ch <- prometheus.MustNewConstMetric(
		bytesDesc,
		prometheus.CounterValue,
		float64(lanInfo.BytesOut),
		"out",
	)

	// Packet metrics
	ch <- prometheus.MustNewConstMetric(
		packetsDesc,
		prometheus.CounterValue,
		float64(lanInfo.PacketsUnicastIn),
		"in",
		"unicast",
	)
	ch <- prometheus.MustNewConstMetric(
		packetsDesc,
		prometheus.CounterValue,
		float64(lanInfo.PacketsUnicastOut),
		"out",
		"unicast",
	)
	ch <- prometheus.MustNewConstMetric(
		packetsDesc,
		prometheus.CounterValue,
		float64(lanInfo.PacketsMulticastIn),
		"in",
		"multicast",
	)
	ch <- prometheus.MustNewConstMetric(
		packetsDesc,
		prometheus.CounterValue,
		float64(lanInfo.PacketsMulticastOut),
		"out",
		"multicast",
	)

	// Error metrics
	ch <- prometheus.MustNewConstMetric(
		errorsDesc,
		prometheus.CounterValue,
		float64(lanInfo.PacketsErrorIn),
		"in",
	)
	ch <- prometheus.MustNewConstMetric(
		errorsDesc,
		prometheus.CounterValue,
		float64(lanInfo.PacketsErrorOut),
		"out",
	)

	// Discard metrics
	ch <- prometheus.MustNewConstMetric(
		discardsDesc,
		prometheus.CounterValue,
		float64(lanInfo.PacketsDiscardedIn),
		"in",
	)
	ch <- prometheus.MustNewConstMetric(
		discardsDesc,
		prometheus.CounterValue,
		float64(lanInfo.PacketsDiscardedOut),
		"out",
	)

	// Status metric
	ch <- prometheus.MustNewConstMetric(
		networkStatusDesc,
		prometheus.GaugeValue,
		float64(lanInfo.Status),
		strconv.Itoa(lanInfo.Speed),
		lanInfo.Duplex,
	)

	// Collect Optical Info
	opticalInfo, err := loadWithRelogin(c, (*ont.Session).LoadOpticalData)
	if err != nil {
		log.Printf("Unable to collect optical info: %v", err)
		return
	}
	// Signal power metrics
	ch <- prometheus.MustNewConstMetric(
		opticalSignalDesc,
		prometheus.GaugeValue,
		opticalInfo.TXPower,
		"tx",
	)
	ch <- prometheus.MustNewConstMetric(
		opticalSignalDesc,
		prometheus.GaugeValue,
		opticalInfo.RXPower,
		"rx",
	)

	// Temperature metric
	ch <- prometheus.MustNewConstMetric(
		opticalTempDesc,
		prometheus.GaugeValue,
		opticalInfo.OpticalModuleTemperature,
	)

	// Voltage metric
	ch <- prometheus.MustNewConstMetric(
		opticalVoltageDesc,
		prometheus.GaugeValue,
		float64(opticalInfo.OpticalModuleVoltage),
	)

	// Bias current metric
	ch <- prometheus.MustNewConstMetric(
		opticalBiasCurrentDesc,
		prometheus.GaugeValue,
		opticalInfo.OpticalModuleBiasCurrent,
	)

	// Status metrics
	ch <- prometheus.MustNewConstMetric(
		opticalStatusDesc,
		prometheus.GaugeValue,
		float64(opticalInfo.LoS),
		"los",
	)
	ch <- prometheus.MustNewConstMetric(
		opticalStatusDesc,
		prometheus.GaugeValue,
		float64(opticalInfo.GPONRegistrationStatus),
		"gpon_registration",
	)
	ch <- prometheus.MustNewConstMetric(
		opticalStatusDesc,
		prometheus.GaugeValue,
		float64(opticalInfo.PONCatV),
		"catv",
	)

	// RF power metrics
	ch <- prometheus.MustNewConstMetric(
		rfPowerDesc,
		prometheus.GaugeValue,
		float64(opticalInfo.RFTXPower),
		"tx",
	)
	ch <- prometheus.MustNewConstMetric(
		rfPowerDesc,
		prometheus.GaugeValue,
		float64(opticalInfo.VideoRXPower),
		"rx",
	)
	ch <- prometheus.MustNewConstMetric(
		opticalUptimeDesc,
		prometheus.CounterValue,
		float64(opticalInfo.Uptime),
	)
}
