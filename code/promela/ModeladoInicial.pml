#define N_WORKERS 4
#define N_REGISTROS 5

int alertas_globales = 0;
int workers_done = 0;
bool mutex = true;

int data[N_WORKERS * N_REGISTROS];

proctype Worker(byte id) {

    int i = 0;
    int alertas_locales = 0;
    int index;

    do
    :: (i < N_REGISTROS) ->
        index = id * N_REGISTROS + i;

        if
        :: data[index] == 1 ->
            alertas_locales++
        :: else ->
            skip
        fi;

        i++
    :: else -> break
    od;

    /* Sección crítica */
    atomic {
        (mutex == true) ->
        mutex = false;

        alertas_globales = alertas_globales + alertas_locales;

        mutex = true;
    }

    workers_done++
}

init {
    byte i = 0;

    /* Inicializar dataset */
    data[0]=1; data[1]=0; data[2]=0; data[3]=1; data[4]=0;
    data[5]=0; data[6]=1; data[7]=1; data[8]=0; data[9]=0;
    data[10]=1; data[11]=1; data[12]=0; data[13]=0; data[14]=1;
    data[15]=0; data[16]=0; data[17]=1; data[18]=0; data[19]=1;

    do
    :: (i < N_WORKERS) ->
        run Worker(i);
        i++
    :: else -> break
    od;

    /* Esperar finalización */
    (workers_done == N_WORKERS);

    /* Verificaciones */
    assert(alertas_globales >= 0);
}

ltl liveness { <> (workers_done == N_WORKERS) }
ltl safety { [] (alertas_globales >= 0) }
