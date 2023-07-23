package server

import (
	"github.com/gorilla/mux"
	"github.com/myntra/goscheduler/constants"
	"github.com/myntra/goscheduler/service"
	"log"
	"net/http"
)

type Server struct {
	port    string
	router  *mux.Router
	service *service.Service
}

func NewHTTPServer(port string, router *mux.Router, service *service.Service) {
	server := &Server{
		port:    port,
		router:  router,
		service: service,
	}
	server.registerHTTPHandlers()
}

func responseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) monitoringMiddleware(operationName string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.service.Monitoring != nil && s.service.Monitoring.StatsDClient != nil {
			s.service.Monitoring.StatsDClient.Increment(operationName)
			timing := s.service.Monitoring.StatsDClient.NewTiming()
			defer timing.Send(operationName)
		}

		if s.service.Monitoring != nil && s.service.Monitoring.NewrelicApp != nil {
			txn := (*s.service.Monitoring.NewrelicApp).StartTransaction(operationName, w, r)
			defer txn.End()
		}

		// Call the next middleware or route handler in the chain
		next(w, r)
	}
}

func (s *Server) registerHTTPHandlers() {
	s.router.Use(responseMiddleware)

	s.router.HandleFunc("/goscheduler/healthcheck", service.HealthCheck)

	s.router.HandleFunc("/goscheduler/schedules",
		s.monitoringMiddleware(constants.CreateSchedule, func(w http.ResponseWriter, r *http.Request) {
			s.service.Post(w, r)
		}),
	).Methods("POST")

	s.router.HandleFunc("/goscheduler/schedules/{scheduleId}",
		s.monitoringMiddleware(constants.GetSchedule, func(w http.ResponseWriter, r *http.Request) {
			s.service.Get(w, r)
		}),
	).Methods("GET")

	s.router.HandleFunc("/goscheduler/schedules/{scheduleId}/runs",
		s.monitoringMiddleware(constants.GetScheduleRuns, func(w http.ResponseWriter, r *http.Request) {
			s.service.GetRuns(w, r)
		}),
	).Methods("GET")

	s.router.HandleFunc("/goscheduler/apps/{appId}/schedules",
		s.monitoringMiddleware(constants.GetAppSchedule, func(w http.ResponseWriter, r *http.Request) {
			s.service.GetAppSchedules(w, r)
		}),
	).Methods("GET")

	s.router.HandleFunc("/goscheduler/schedules/{scheduleId}",
		s.monitoringMiddleware(constants.DeleteSchedule, func(w http.ResponseWriter, r *http.Request) {
			s.service.CancelSchedule(w, r)
		}),
	).Methods("DELETE")

	s.router.HandleFunc("/goscheduler/apps",
		s.monitoringMiddleware(constants.RegisterApp, func(w http.ResponseWriter, r *http.Request) {
			s.service.Register(w, r)
		}),
	).Methods("POST")

	s.router.HandleFunc("/goscheduler/apps/{appId}/deactivate",
		s.monitoringMiddleware(constants.DeactivateApp, func(w http.ResponseWriter, r *http.Request) {
			s.service.Deactivate(w, r)
		}),
	).Methods("POST")

	s.router.HandleFunc("/goscheduler/apps/{appId}/activate",
		s.monitoringMiddleware(constants.ActivateApp, func(w http.ResponseWriter, r *http.Request) {
			s.service.Activate(w, r)
		}),
	).Methods("POST")

	s.router.HandleFunc("/goscheduler/apps/{appId}/bulk-action/{action}",
		s.monitoringMiddleware(constants.BulkAction, func(w http.ResponseWriter, r *http.Request) {
			s.service.BulkAction(w, r)
		}),
	).Methods("POST")

	s.router.HandleFunc("/goscheduler/apps",
		s.monitoringMiddleware(constants.GetApps, func(w http.ResponseWriter, r *http.Request) {
			s.service.GetApps(w, r)
		}),
	).Methods("GET")

	s.router.HandleFunc("/goscheduler/crons/schedules",
		s.monitoringMiddleware(constants.GetCronSchedule, func(w http.ResponseWriter, r *http.Request) {
			s.service.GetCronSchedules(w, r)
		}),
	).Methods("GET")

	log.Fatal(http.ListenAndServe(":"+s.port, s.router))
}